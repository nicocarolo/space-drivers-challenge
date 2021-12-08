package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nicocarolo/space-drivers/internal/platform/jwt"
	"github.com/nicocarolo/space-drivers/internal/travel"
	"github.com/nicocarolo/space-drivers/internal/user"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

// travelMockDb a 'db' to use on TravelStorage test with the capabilities to mock errors on create/get/update action
type travelMockDb struct {
	idCount int64
	travels map[int64]travel.Travel

	saveError   error
	getError    map[int64]error
	updateError map[int64]error
}

func (db *travelMockDb) onCreate(err error) *travelMockDb {
	db.saveError = err

	return db
}

func (db *travelMockDb) onGet(id int64, err error) *travelMockDb {
	db.getError[id] = err

	return db
}

func (db *travelMockDb) onUpdate(id int64, err error) *travelMockDb {
	db.updateError[id] = err

	return db
}

func (db *travelMockDb) SaveTravel(ctx context.Context, trv travel.Travel) (travel.Travel, error) {
	if db.saveError != nil {
		err := db.saveError
		db.saveError = nil
		return travel.Travel{}, err
	}

	trv.ID = db.idCount
	db.travels[trv.ID] = trv

	db.idCount++

	return trv, nil
}

func (db travelMockDb) GetTravel(ctx context.Context, id int64) (travel.Travel, error) {
	if err, ok := db.getError[id]; ok {
		return travel.Travel{}, err
	}

	trv, exist := db.travels[id]
	if !exist {
		return travel.Travel{}, fmt.Errorf("not found travel")
	}

	return trv, nil
}

func (db *travelMockDb) EditTravel(ctx context.Context, newTravel travel.Travel) error {
	if err, ok := db.updateError[newTravel.ID]; ok {
		return err
	}
	_, exist := db.travels[newTravel.ID]
	if !exist {
		return fmt.Errorf("not found travel")
	}

	db.travels[newTravel.ID] = newTravel

	return nil
}

func newTravelMockDb() *travelMockDb {
	return &travelMockDb{
		idCount: 1,
		travels: make(map[int64]travel.Travel),

		getError:    make(map[int64]error),
		updateError: make(map[int64]error),
	}
}

func newTravelMockDbFromMap(travels map[int64]travel.Travel) *travelMockDb {
	return &travelMockDb{
		idCount: 1,
		travels: travels,

		getError:    make(map[int64]error),
		updateError: make(map[int64]error),
	}
}

func Test_createTravel(t *testing.T) {
	testscases := map[string]struct {
		travelStorage  TravelStorage
		body           map[string]interface{}
		want           travel.Travel
		wantError      error
		statusExpected int
	}{
		"successful created travel without user and status": {
			travelStorage: travel.NewTravelStorage(newTravelMockDb()),
			body: map[string]interface{}{
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			want: travel.Travel{
				ID:     1,
				Status: "pending",
				From: travel.Point{
					Lat: 1,
					Lng: 2,
				},
				To: travel.Point{
					Lat: -1,
					Lng: -2,
				},
				UserID: 0,
			},
			statusExpected: http.StatusCreated,
		},

		"successful created travel without user and with status": {
			travelStorage: travel.NewTravelStorage(newTravelMockDb()),
			body: map[string]interface{}{
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
				"status": "pending",
			},
			want: travel.Travel{
				ID:     1,
				Status: "pending",
				From: travel.Point{
					Lat: 1,
					Lng: 2,
				},
				To: travel.Point{
					Lat: -1,
					Lng: -2,
				},
				UserID: 0,
			},
			statusExpected: http.StatusCreated,
		},

		"successful created travel with user and without status": {
			travelStorage: travel.NewTravelStorage(newTravelMockDb()),
			body: map[string]interface{}{
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
				"user_id": 1,
			},
			want: travel.Travel{
				ID:     1,
				Status: "pending",
				From: travel.Point{
					Lat: 1,
					Lng: 2,
				},
				To: travel.Point{
					Lat: -1,
					Lng: -2,
				},
				UserID: 1,
			},
			statusExpected: http.StatusCreated,
		},

		"successful created travel with status different than pending": {
			travelStorage: travel.NewTravelStorage(newTravelMockDb()),
			body: map[string]interface{}{
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
				"status": "in_process",
			},
			want: travel.Travel{
				ID:     1,
				Status: "pending",
				From: travel.Point{
					Lat: 1,
					Lng: 2,
				},
				To: travel.Point{
					Lat: -1,
					Lng: -2,
				},
			},
			statusExpected: http.StatusCreated,
		},

		"failure due to invalid request: no from": {
			travelStorage: travel.NewTravelStorage(newTravelMockDb()),
			body: map[string]interface{}{
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
			},
			wantError:      errors.New("invalid_request - there was an error with fields: lat,lng"),
			statusExpected: http.StatusUnprocessableEntity,
		},

		"failure due to invalid request: no to": {
			travelStorage: travel.NewTravelStorage(newTravelMockDb()),
			body: map[string]interface{}{
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("invalid_request - there was an error with fields: lat,lng"),
			statusExpected: http.StatusUnprocessableEntity,
		},

		"failure due to storage failure": {
			travelStorage: travel.NewTravelStorage(newTravelMockDb().onCreate(errors.New("mocked storage error"))),
			body: map[string]interface{}{
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("storage_failure - an error ocurred trying to save travel"),
			statusExpected: http.StatusInternalServerError,
		},
	}

	for name, tc := range testscases {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = &http.Request{
				Header: make(http.Header),
			}

			err := mockJson(c, http.MethodPost, tc.body)
			assert.Nil(t, err)

			handler := TravelHandler{
				Travels: tc.travelStorage,
			}
			handler.Create(c)

			assert.Equal(t, tc.statusExpected, w.Code)

			if tc.wantError != nil {
				var apiErr apiError
				err = json.Unmarshal(w.Body.Bytes(), &apiErr)
				assert.Nil(t, err)

				assert.Equal(t, tc.wantError.Error(), apiErr.Error())
			} else {
				response := travel.Travel{}

				err = json.Unmarshal(w.Body.Bytes(), &response)
				assert.Nil(t, err)

				assert.Equal(t, tc.want.From.Lat, response.From.Lat)
				assert.Equal(t, tc.want.From.Lng, response.From.Lng)
				assert.Equal(t, tc.want.Status, response.Status)
				assert.Equal(t, tc.want.UserID, response.UserID)
				assert.Greater(t, response.ID, int64(0))
			}
		})
	}
}

func Test_getTravel(t *testing.T) {
	dbWithUser := newTravelMockDb()
	_, _ = dbWithUser.SaveTravel(context.Background(), travel.Travel{
		ID:     1,
		Status: "pending",
		From: travel.Point{
			Lat: 1,
			Lng: 2,
		},
		To: travel.Point{
			Lat: -1,
			Lng: -2,
		},
		UserID: 1,
	})

	createURLParam := func(id string) []gin.Param {
		return []gin.Param{
			{
				Key:   "id",
				Value: id,
			},
		}
	}

	testscases := map[string]struct {
		travelStorage  TravelStorage
		urlParam       []gin.Param
		want           travel.Travel
		wantError      error
		statusExpected int
	}{
		"successful get travel": {
			travelStorage: travel.NewTravelStorage(dbWithUser),
			urlParam:      createURLParam("1"),
			want: travel.Travel{
				ID:     1,
				Status: "pending",
				From: travel.Point{
					Lat: 1,
					Lng: 2,
				},
				To: travel.Point{
					Lat: -1,
					Lng: -2,
				},
				UserID: 1,
			},
			statusExpected: http.StatusOK,
		},

		"failure due to invalid request: no id": {
			travelStorage:  travel.NewTravelStorage(newTravelMockDb()),
			wantError:      errors.New("invalid_request - the request has not a travel id to get"),
			statusExpected: http.StatusBadRequest,
		},

		"failure due to storage error": {
			travelStorage:  travel.NewTravelStorage(newTravelMockDb().onGet(2, travel.ErrStorageGet)),
			urlParam:       createURLParam("1"),
			wantError:      errors.New("storage_failure - an error ocurred trying to get travel"),
			statusExpected: http.StatusInternalServerError,
		},

		"failure due to non existent travel": {
			travelStorage:  travel.NewTravelStorage(newTravelMockDb().onGet(4, travel.ErrTravelNotFound)),
			urlParam:       createURLParam("4"),
			wantError:      errors.New("not_found_travel - not founded the travel to get"),
			statusExpected: http.StatusNotFound,
		},
	}

	for name, tc := range testscases {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = &http.Request{
				Header: make(http.Header),
			}

			c.Params = tc.urlParam

			handler := TravelHandler{
				Travels: tc.travelStorage,
			}
			handler.Get(c)

			assert.Equal(t, tc.statusExpected, w.Code)

			if tc.wantError != nil {
				var apiErr apiError
				err := json.Unmarshal(w.Body.Bytes(), &apiErr)
				assert.Nil(t, err)

				assert.Equal(t, tc.wantError.Error(), apiErr.Error())
			} else {
				response := travel.Travel{}

				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.Nil(t, err)

				assert.Equal(t, tc.want.From.Lat, response.From.Lat)
				assert.Equal(t, tc.want.From.Lng, response.From.Lng)
				assert.Equal(t, tc.want.Status, response.Status)
				assert.Equal(t, tc.want.UserID, response.UserID)
				assert.Greater(t, response.ID, int64(0))
			}
		})
	}
}

func Test_editTravel(t *testing.T) {
	userDB := newMockDB()
	_, _ = userDB.SaveUser(context.Background(), user.User{
		SecuredUser: user.SecuredUser{
			ID:    1,
			Email: "an_email@hotmail.com",
			Role:  "driver",
		},
	})
	_, _ = userDB.SaveUser(context.Background(), user.User{
		SecuredUser: user.SecuredUser{
			ID:    2,
			Email: "another_email@hotmail.com",
			Role:  "driver",
		},
	})
	_, _ = userDB.SaveUser(context.Background(), user.User{
		SecuredUser: user.SecuredUser{
			ID:    5,
			Email: "another2_email@hotmail.com",
			Role:  "driver",
		},
	})
	userDB = userDB.onGet(3, user.ErrUserNotFound)
	storageWithUser := user.NewUserStorage(userDB)

	newTravel := func(id int64, fromLat, fromLng, toLat, toLng float64, status travel.Status, userID int64) travel.Travel {
		return travel.Travel{
			ID:     id,
			Status: status,
			From: travel.Point{
				Lat: fromLat,
				Lng: fromLng,
			},
			To: travel.Point{
				Lat: toLat,
				Lng: toLng,
			},
			UserID: userID,
		}
	}

	createURLParam := func(id string) []gin.Param {
		return []gin.Param{
			{
				Key:   "id",
				Value: id,
			},
		}
	}

	testscases := map[string]struct {
		travelStorage  TravelStorage
		urlParam       []gin.Param
		userLogged     *jwt.Claims
		body           map[string]interface{}
		want           travel.Travel
		wantError      error
		statusExpected int
	}{
		"successful edit travel without user: change location": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, -100, 70, 2, 20, travel.StatusPending, 0)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"status": "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			want: travel.Travel{
				ID:     1,
				Status: "pending",
				From: travel.Point{
					Lat: 1,
					Lng: 2,
				},
				To: travel.Point{
					Lat: -1,
					Lng: -2,
				},
				UserID: 0,
			},
			statusExpected: http.StatusOK,
		},

		"successful edit travel with user: change location": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, -100, 70, 2, 20, travel.StatusPending, 0)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"user_id": 1,
				"status":  "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			want: travel.Travel{
				ID:     1,
				Status: "pending",
				From: travel.Point{
					Lat: 1,
					Lng: 2,
				},
				To: travel.Point{
					Lat: -1,
					Lng: -2,
				},
				UserID: 1,
			},
			statusExpected: http.StatusOK,
		},

		"successful edit travel with user: change status": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusPending, 0)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"user_id": 1,
				"status":  "in_process",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			want: travel.Travel{
				ID:     1,
				Status: "in_process",
				From: travel.Point{
					Lat: 1,
					Lng: 2,
				},
				To: travel.Point{
					Lat: -1,
					Lng: -2,
				},
				UserID: 1,
			},
			statusExpected: http.StatusOK,
		},

		"successful edit travel by admin with one user to another user due to it is in pending status": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusPending, 2)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"user_id": 5,
				"status":  "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			want: travel.Travel{
				ID:     1,
				Status: "pending",
				From: travel.Point{
					Lat: 1,
					Lng: 2,
				},
				To: travel.Point{
					Lat: -1,
					Lng: -2,
				},
				UserID: 5,
			},
			statusExpected: http.StatusOK,
		},

		"failure edit travel by driver with one user to another user, no matter if it is pending": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusPending, 2)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "driver",
			},
			body: map[string]interface{}{
				"user_id": 5,
				"status":  "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("invalid_user_access - the user logged in cannot perform this action, he is not the owner of the travel or it is not an admin"),
			statusExpected: http.StatusUnauthorized,
		},

		"failure travel update: change initial status without user on db travel": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusPending, 0)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"status": "in_process",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("invalid_user - invalid user while performing update"),
			statusExpected: http.StatusBadRequest,
		},

		"failure travel update: change locations in no pending status": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusPending, 0)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"status": "in_process",
				"from": map[string]float64{
					"latitude":  100,
					"longitude": 200,
				},
				"to": map[string]float64{
					"latitude":  -100,
					"longitude": -200,
				},
			},
			wantError:      errors.New("invalid_user - invalid user while performing update"),
			statusExpected: http.StatusBadRequest,
		},

		"failure travel update: change user id in no pending status": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusInProcess, 1)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 2,
				Role:   "driver",
			},
			body: map[string]interface{}{
				"user_id": 2,
				"status":  "in_process",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("invalid_user_access - the user logged in cannot perform this action, he is not the owner of the travel or it is not an admin"),
			statusExpected: http.StatusUnauthorized,
		},

		"failure travel update: no user id in no pending status": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusInProcess, 2)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"status": "in_process",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("invalid_user - invalid user while performing update"),
			statusExpected: http.StatusBadRequest,
		},

		"failure travel update: pending to ready": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusPending, 1)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"user_id": 1,
				"status":  "ready",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("invalid_status - invalid received status"),
			statusExpected: http.StatusBadRequest,
		},

		"failure travel update: in process to pending": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusInProcess, 1)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"user_id": 1,
				"status":  "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("invalid_status - invalid received status"),
			statusExpected: http.StatusBadRequest,
		},

		"failure due to invalid request: no id": {
			travelStorage:  travel.NewTravelStorage(newTravelMockDb()),
			wantError:      errors.New("invalid_request - the request has not a travel id to update"),
			statusExpected: http.StatusBadRequest,
		},

		"failure due to invalid request: no location": {
			travelStorage: travel.NewTravelStorage(newTravelMockDb()),
			urlParam:      createURLParam("1"),
			body: map[string]interface{}{
				"status": "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
			},
			wantError:      errors.New("invalid_request - there was an error with fields: lat,lng"),
			statusExpected: http.StatusUnprocessableEntity,
		},

		"failure due to storage error on get": {
			travelStorage: travel.NewTravelStorage(newTravelMockDb().onGet(1, travel.ErrStorageGet)),
			urlParam:      createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"user_id": 1,
				"status":  "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("storage_failure - an error ocurred trying to get travel"),
			statusExpected: http.StatusInternalServerError,
		},

		"failure due to storage error on update": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusPending, 1)}).
				onUpdate(1, travel.ErrStorageUpdate)),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"user_id": 1,
				"status":  "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("storage_failure - an error ocurred trying to update travel"),
			statusExpected: http.StatusInternalServerError,
		},

		"failure due to non existent travel": {
			travelStorage: travel.NewTravelStorage(newTravelMockDb().onGet(4, travel.ErrTravelNotFound)),
			urlParam:      createURLParam("4"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"user_id": 1,
				"status":  "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("not_found_travel - not founded the travel to get"),
			statusExpected: http.StatusNotFound,
		},

		"failure due to non existent user": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusInProcess, 1)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "admin",
			},
			body: map[string]interface{}{
				"user_id": 3,
				"status":  "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("invalid_travel_user - the user received was not found"),
			statusExpected: http.StatusBadRequest,
		},

		"failure due to non user logged in": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusInProcess, 1)})),
			urlParam: createURLParam("1"),
			body: map[string]interface{}{
				"user_id": 1,
				"status":  "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("invalid_user_access - cannot identify user logged in"),
			statusExpected: http.StatusUnauthorized,
		},

		"failure due to the user who is logged in is not an admin and is not the owner of the travel": {
			travelStorage: travel.NewTravelStorage(newTravelMockDbFromMap(map[int64]travel.Travel{
				1: newTravel(1, 1, 2, -1, -2, travel.StatusInProcess, 2)})),
			urlParam: createURLParam("1"),
			userLogged: &jwt.Claims{
				UserID: 1,
				Role:   "driver",
			},
			body: map[string]interface{}{
				"user_id": 1,
				"status":  "pending",
				"from": map[string]float64{
					"latitude":  1,
					"longitude": 2,
				},
				"to": map[string]float64{
					"latitude":  -1,
					"longitude": -2,
				},
			},
			wantError:      errors.New("invalid_user_access - the user logged in cannot perform this action, he is not the owner of the travel or it is not an admin"),
			statusExpected: http.StatusUnauthorized,
		},
	}

	for name, tc := range testscases {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = &http.Request{
				Header: make(http.Header),
			}

			err := mockJson(c, http.MethodPost, tc.body)
			assert.Nil(t, err)
			c.Params = tc.urlParam

			if tc.userLogged != nil {
				c.Set("user_on_call", *tc.userLogged)
			}

			handler := TravelHandler{
				Travels: tc.travelStorage,
				Users:   storageWithUser,
			}
			handler.Edit(c)

			assert.Equal(t, tc.statusExpected, w.Code)

			if tc.wantError != nil {
				var apiErr apiError
				err := json.Unmarshal(w.Body.Bytes(), &apiErr)
				assert.Nil(t, err)

				assert.Equal(t, tc.wantError.Error(), apiErr.Error())
			} else {
				response := travel.Travel{}

				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.Nil(t, err)

				assert.Equal(t, tc.want.From.Lat, response.From.Lat)
				assert.Equal(t, tc.want.From.Lng, response.From.Lng)
				assert.Equal(t, tc.want.Status, response.Status)
				assert.Equal(t, tc.want.UserID, response.UserID)
				assert.Greater(t, response.ID, int64(0))
			}
		})
	}
}
