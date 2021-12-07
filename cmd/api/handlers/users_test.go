package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nicocarolo/space-drivers/internal/user"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

type FailureEncrypter struct{}

func (f FailureEncrypter) Encrypt(pwd string) ([]byte, error) {
	return nil, fmt.Errorf("mocked password crypto error")
}

func (f FailureEncrypter) Compare(encrypted, pwd string) error {
	return fmt.Errorf("mocked password crypto error")
}

// mockDb a 'db' to use on UserHandler test with the capabilities to mock errors on create/get action
type mockDb struct {
	idCount int64
	users   map[int64]user.User

	saveError           map[string]error
	getError            map[int64]error
	getFreeDriversError error
}

func newMockDB() *mockDb {
	return &mockDb{
		idCount: 1,
		users:   make(map[int64]user.User),

		saveError: make(map[string]error),
		getError:  make(map[int64]error),
	}
}

func (db *mockDb) onCreate(email string, err error) *mockDb {
	db.saveError[email] = err
	return db
}

func (db *mockDb) onGet(id int64, err error) *mockDb {
	db.getError[id] = err
	return db
}

func (db *mockDb) onGetFreeDrivers(err error) *mockDb {
	db.getFreeDriversError = err
	return db
}

func (db *mockDb) SaveUser(ctx context.Context, u user.User) (user.User, error) {
	if err, ok := db.saveError[u.Email]; ok {
		return user.User{}, err
	}

	u.ID = db.idCount
	db.users[u.ID] = u

	db.idCount++

	return u, nil
}

func (db mockDb) GetUser(ctx context.Context, id int64) (user.User, error) {
	if err, ok := db.getError[id]; ok {
		return user.User{}, err
	}

	u, exist := db.users[id]
	if !exist {
		return user.User{}, fmt.Errorf("not found user")
	}

	return u, nil
}

func (db mockDb) GetUserByEmail(ctx context.Context, email string) (user.User, error) {
	for _, u := range db.users {
		if u.Email == email {
			return u, nil
		}
	}
	return user.User{}, user.ErrUserNotFound
}

func (db mockDb) GetFreeDrivers(ctx context.Context) ([]user.User, error) {
	if db.getFreeDriversError != nil {
		return nil, db.getFreeDriversError
	}
	return []user.User{
		user.User{
			SecuredUser: user.SecuredUser{
				ID:    1,
				Email: "an_email@hotmail.com",
				Role:  "driver",
			},
		},
		user.User{
			SecuredUser: user.SecuredUser{
				ID:    2,
				Email: "another_email@hotmail.com",
				Role:  "driver",
			},
		},
	}, nil
}

func (db mockDb) GetPaginate(ctx context.Context, limit, offset int64) ([]user.User, int64, error) {
	users := []user.User{
		user.User{
			SecuredUser: user.SecuredUser{
				ID:    1,
				Email: "an_email@hotmail.com",
				Role:  "driver",
			},
		},
		user.User{
			SecuredUser: user.SecuredUser{
				ID:    2,
				Email: "another_email@hotmail.com",
				Role:  "driver",
			},
		},
		user.User{
			SecuredUser: user.SecuredUser{
				ID:    3,
				Email: "another_email3@hotmail.com",
				Role:  "driver",
			},
		},
	}

	top := int64(len(users))
	if limit < top {
		top = limit
	}
	return users[offset:top], int64(len(users)), nil
}

func Test_createUser(t *testing.T) {
	testscases := map[string]struct {
		userStorage    UsersStorage
		body           map[string]interface{}
		want           user.SecuredUser
		wantError      error
		statusExpected int
	}{
		"successful created user": {
			userStorage: user.NewUserStorage(newMockDB()),
			body: map[string]interface{}{
				"email":    "a user email",
				"password": "a user pass",
				"role":     "driver",
			},
			want: user.SecuredUser{
				Email: "a user email",
				Role:  "driver",
			},
			statusExpected: http.StatusCreated,
		},

		"failure due to invalid request: no password": {
			userStorage: user.NewUserStorage(newMockDB()),
			body: map[string]interface{}{
				"email": "a user email",
				"role":  "driver",
			},
			wantError:      errors.New("invalid_request - there was an error with fields: password"),
			statusExpected: http.StatusUnprocessableEntity,
		},

		"failure due to invalid password": {
			userStorage: user.NewUserStorage(newMockDB(), user.WithPasswordEncrypter(FailureEncrypter{})),
			body: map[string]interface{}{
				"email":    "a user email",
				"password": "an invalid pass",
				"role":     "driver",
			},
			wantError:      errors.New("invalid_password - cannot assign received password to user"),
			statusExpected: http.StatusBadRequest,
		},

		"failure due to invalid role": {
			userStorage: user.NewUserStorage(newMockDB()),
			body: map[string]interface{}{
				"email":    "a user email",
				"password": "an invalid pass",
				"role":     "an invalid role",
			},
			wantError:      errors.New("invalid_role - the received role should be admin or driver"),
			statusExpected: http.StatusBadRequest,
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

			handler := UserHandler{
				Users: tc.userStorage,
			}
			handler.Create(c)

			assert.Equal(t, tc.statusExpected, w.Code)

			if tc.wantError != nil {
				var apiErr apiError
				err = json.Unmarshal(w.Body.Bytes(), &apiErr)
				assert.Nil(t, err)

				assert.Equal(t, tc.wantError.Error(), apiErr.Error())
			} else {
				response := user.SecuredUser{}

				err = json.Unmarshal(w.Body.Bytes(), &response)
				assert.Nil(t, err)

				assert.Equal(t, tc.want.Email, response.Email)
				assert.Equal(t, tc.want.Role, response.Role)
				assert.Greater(t, response.ID, int64(0))
			}
		})
	}
}

func Test_getUser(t *testing.T) {
	dbWithUser := newMockDB()
	createdUser, _ := dbWithUser.SaveUser(context.Background(), user.User{
		SecuredUser: user.SecuredUser{
			Email: "anEmail@asa.com",
			Role:  "admin",
		},
		Password: "a pass",
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
		userStorage    UsersStorage
		urlParams      gin.Params
		want           user.SecuredUser
		wantError      error
		statusExpected int
	}{
		"successful get user": {
			userStorage: user.NewUserStorage(dbWithUser),
			urlParams:   createURLParam(strconv.FormatInt(createdUser.ID, 10)),
			want: user.SecuredUser{
				ID:    createdUser.ID,
				Email: "anEmail@asa.com",
				Role:  "admin",
			},
			statusExpected: http.StatusOK,
		},

		"failure due to invalid request: no id": {
			userStorage:    user.NewUserStorage(newMockDB()),
			wantError:      errors.New("invalid_request - the request has not a user id to get"),
			statusExpected: http.StatusBadRequest,
		},

		"failure due to non existent user": {
			userStorage:    user.NewUserStorage(newMockDB().onGet(1, user.ErrUserNotFound)),
			urlParams:      createURLParam("1"),
			wantError:      errors.New("not_found_user - not founded the user to get"),
			statusExpected: http.StatusNotFound,
		},
	}

	for name, tc := range testscases {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)

			c.Params = tc.urlParams

			handler := UserHandler{
				Users: tc.userStorage,
			}

			handler.Get(c)

			assert.Equal(t, tc.statusExpected, w.Code)

			if tc.wantError != nil {
				var apiErr apiError
				err := json.Unmarshal(w.Body.Bytes(), &apiErr)
				assert.Nil(t, err)

				assert.Equal(t, tc.wantError.Error(), apiErr.Error())
			} else {
				response := user.SecuredUser{}

				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.Nil(t, err)

				assert.Equal(t, tc.want.Email, response.Email)
				assert.Equal(t, tc.want.Role, response.Role)
				assert.Equal(t, tc.want.ID, response.ID)
			}
		})
	}
}

func Test_searchUser(t *testing.T) {
	type response struct {
		Total   int64              `json:"total"`
		Pending int64              `json:"pending"`
		Result  []user.SecuredUser `json:"result"`
	}

	testscases := map[string]struct {
		userStorage    UsersStorage
		urlParams      map[string]string
		want           response
		wantError      error
		statusExpected int
	}{
		"successful get free drivers": {
			userStorage: user.NewUserStorage(newMockDB()),
			urlParams: map[string]string{
				"status": "free",
			},
			want: response{
				Total:   2,
				Pending: 0,
				Result: []user.SecuredUser{
					user.SecuredUser{
						ID:    1,
						Email: "an_email@hotmail.com",
						Role:  "driver",
					},
					user.SecuredUser{
						ID:    2,
						Email: "another_email@hotmail.com",
						Role:  "driver",
					},
				},
			},
			statusExpected: http.StatusOK,
		},

		"failure get free drivers: bad status": {
			userStorage: user.NewUserStorage(newMockDB()),
			urlParams: map[string]string{
				"status": "unknown",
			},
			wantError:      errors.New("invalid_request - invalid search status received"),
			statusExpected: http.StatusBadRequest,
		},

		"failure get free drivers: with limit": {
			userStorage: user.NewUserStorage(newMockDB()),
			urlParams: map[string]string{
				"status": "free",
				"limit":  "0",
			},
			wantError:      errors.New("invalid_request - search free driver do not accept limit or offset param"),
			statusExpected: http.StatusBadRequest,
		},

		"failure get free drivers: with offset": {
			userStorage: user.NewUserStorage(newMockDB()),
			urlParams: map[string]string{
				"status": "free",
				"offset": "0",
			},
			wantError:      errors.New("invalid_request - search free driver do not accept limit or offset param"),
			statusExpected: http.StatusBadRequest,
		},

		"successful get drivers": {
			userStorage: user.NewUserStorage(newMockDB()),
			urlParams:   map[string]string{},
			want: response{
				Total:   3,
				Pending: 0,
				Result: []user.SecuredUser{
					user.SecuredUser{
						ID:    1,
						Email: "an_email@hotmail.com",
						Role:  "driver",
					},
					user.SecuredUser{
						ID:    2,
						Email: "another_email@hotmail.com",
						Role:  "driver",
					},
					user.SecuredUser{
						ID:    3,
						Email: "another_email3@hotmail.com",
						Role:  "driver",
					},
				},
			},
			statusExpected: http.StatusOK,
		},

		"failure get drivers: invalid limit 0": {
			userStorage: user.NewUserStorage(newMockDB()),
			urlParams: map[string]string{
				"limit": "0",
			},
			wantError:      errors.New("invalid_request - invalid search limit received"),
			statusExpected: http.StatusBadRequest,
		},

		"failure get drivers: invalid limit": {
			userStorage: user.NewUserStorage(newMockDB()),
			urlParams: map[string]string{
				"limit": "a",
			},
			wantError:      errors.New("invalid_request - invalid search limit received"),
			statusExpected: http.StatusBadRequest,
		},

		"failure get drivers: invalid offset": {
			userStorage: user.NewUserStorage(newMockDB()),
			urlParams: map[string]string{
				"offset": "a",
			},
			wantError:      errors.New("invalid_request - invalid search offset received"),
			statusExpected: http.StatusBadRequest,
		},

		"successful get drivers: 0 offset": {
			userStorage: user.NewUserStorage(newMockDB()),
			urlParams: map[string]string{
				"offset": "0",
			},
			want: response{
				Total:   3,
				Pending: 0,
				Result: []user.SecuredUser{
					user.SecuredUser{
						ID:    1,
						Email: "an_email@hotmail.com",
						Role:  "driver",
					},
					user.SecuredUser{
						ID:    2,
						Email: "another_email@hotmail.com",
						Role:  "driver",
					},
					user.SecuredUser{
						ID:    3,
						Email: "another_email3@hotmail.com",
						Role:  "driver",
					},
				},
			},
			statusExpected: http.StatusOK,
		},
	}

	for name, tc := range testscases {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)

			req := &http.Request{
				URL:    &url.URL{},
				Header: make(http.Header), // if you need to test headers
			}
			q := req.URL.Query()
			for k, v := range tc.urlParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req

			handler := UserHandler{
				Users: tc.userStorage,
			}

			handler.GetDrivers(c)

			assert.Equal(t, tc.statusExpected, w.Code)

			if tc.wantError != nil {
				var apiErr apiError
				err := json.Unmarshal(w.Body.Bytes(), &apiErr)
				assert.Nil(t, err)

				assert.Equal(t, tc.wantError.Error(), apiErr.Error())
			} else {
				var response struct {
					Total   int64              `json:"total"`
					Pending int64              `json:"pending"`
					Result  []user.SecuredUser `json:"result"`
				}

				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.Nil(t, err)

				assert.Len(t, response.Result, len(tc.want.Result))
				assert.Equal(t, tc.want.Total, response.Total)
				assert.Equal(t, tc.want.Pending, response.Pending)
				for i, securedUser := range tc.want.Result {
					assert.Equal(t, securedUser.ID, response.Result[i].ID)
					assert.Equal(t, securedUser.Email, response.Result[i].Email)
					assert.Equal(t, securedUser.Email, response.Result[i].Email)
					assert.Equal(t, securedUser.Role, response.Result[i].Role)
				}
			}
		})
	}
}

func mockJson(c *gin.Context, method string, body interface{}) error {
	c.Request.Method = method
	c.Request.Header.Set("Content-Type", "application/json")

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(jsonBody))

	return nil
}
