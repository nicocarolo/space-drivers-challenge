package travel

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

// mockDb a 'db' to use on TravelStorage test with the capabilities to mock errors on create/get/update action
type mockDb struct {
	idCount int64
	travels map[int64]Travel

	saveError   error
	getError    map[int64]error
	updateError map[int64]error
}

func (db *mockDb) onCreate(err error) *mockDb {
	db.saveError = err

	return db
}

func (db *mockDb) onGet(id int64, err error) *mockDb {
	db.getError[id] = err

	return db
}

func (db *mockDb) onUpdate(id int64, err error) *mockDb {
	db.updateError[id] = err

	return db
}

func (db *mockDb) SaveTravel(ctx context.Context, travel Travel) (Travel, error) {
	if db.saveError != nil {
		err := db.saveError
		db.saveError = nil
		return Travel{}, err
	}

	travel.ID = db.idCount
	db.travels[travel.ID] = travel

	db.idCount++

	return travel, nil
}

func (db mockDb) GetTravel(ctx context.Context, id int64) (Travel, error) {
	if err, ok := db.getError[id]; ok {
		return Travel{}, err
	}

	travel, exist := db.travels[id]
	if !exist {
		return Travel{}, fmt.Errorf("not found travel")
	}

	return travel, nil
}

func (db *mockDb) EditTravel(ctx context.Context, newTravel Travel) error {
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

func newMockDB() *mockDb {
	return &mockDb{
		idCount: 1,
		travels: make(map[int64]Travel),

		getError:    make(map[int64]error),
		updateError: make(map[int64]error),
	}
}

func newMockDBFromMap(travels map[int64]Travel) *mockDb {
	return &mockDb{
		idCount: 1,
		travels: travels,

		getError:    make(map[int64]error),
		updateError: make(map[int64]error),
	}
}

func Test_createTravel(t *testing.T) {
	tests := map[string]struct {
		db       repository
		trv      Travel
		expected error
	}{
		"successful travel save": {
			db: newMockDB(),
			trv: Travel{
				From: Point{
					Lat: -1,
					Lng: -10,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
				UserID: 121386719,
			},
		},

		"successful travel save without user id": {
			db: newMockDB(),
			trv: Travel{
				From: Point{
					Lat: -1,
					Lng: -10,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
			},
		},

		"successful travel save with status": {
			db: newMockDB(),
			trv: Travel{
				From: Point{
					Lat: -1,
					Lng: -10,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
				Status: "an status",
			},
		},

		"db failure on travel save": {
			db: newMockDB().onCreate(fmt.Errorf("mock db save error")),
			trv: Travel{
				From: Point{
					Lat: -1,
					Lng: -10,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
			},
			expected: ErrStorageSave,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			travelStorage := NewTravelStorage(tc.db)
			result, err := travelStorage.Save(context.Background(), tc.trv)

			if tc.expected == nil {
				assert.Nil(t, err)
				assert.NotNil(t, result)

				assert.Equal(t, StatusPending, string(result.Status))
				assert.Equal(t, tc.trv.From.Lat, result.From.Lat)
				assert.Equal(t, tc.trv.From.Lng, result.From.Lng)
				assert.Equal(t, tc.trv.To.Lat, result.To.Lat)
				assert.Equal(t, tc.trv.To.Lng, result.To.Lng)
				assert.Equal(t, tc.trv.UserID, result.UserID)
				assert.Greater(t, result.ID, int64(0))
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expected.Error(), err.Error())
			}
		})
	}
}

func Test_getTravel(t *testing.T) {
	dbWithUser := newMockDBFromMap(map[int64]Travel{
		1: Travel{
			ID: 1,
			From: Point{
				Lat: -1,
				Lng: -10,
			},
			To: Point{
				Lat: 2,
				Lng: 20,
			},
		},
		2: Travel{
			ID: 2,
			From: Point{
				Lat: -1,
				Lng: -10,
			},
			To: Point{
				Lat: 2,
				Lng: 20,
			},
			Status: StatusReady,
			UserID: 23456789,
		},
	})

	tests := map[string]struct {
		db       repository
		id       int64
		want     Travel
		expected error
	}{
		"successful travel get": {
			db: dbWithUser,
			id: 1,
			want: Travel{
				From: Point{
					Lat: -1,
					Lng: -10,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
			},
		},

		"successful travel with status and user get": {
			db: dbWithUser,
			id: 2,
			want: Travel{
				From: Point{
					Lat: -1,
					Lng: -10,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
				Status: StatusReady,
				UserID: 23456789,
			},
		},

		"db failure travel not found": {
			db:       newMockDB().onGet(22, ErrTravelNotFound),
			id:       22,
			expected: ErrNotFoundTravel,
		},

		"db failure user get": {
			db:       newMockDB().onGet(22, errors.New("mocked get error")),
			id:       22,
			expected: ErrStorageGet,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			travelStorage := NewTravelStorage(tc.db)
			result, err := travelStorage.Get(context.Background(), tc.id)

			if tc.expected == nil {
				assert.Nil(t, err)
				assert.NotNil(t, result)

				assert.Equal(t, tc.want.Status, result.Status)
				assert.Equal(t, tc.want.From.Lat, result.From.Lat)
				assert.Equal(t, tc.want.From.Lng, result.From.Lng)
				assert.Equal(t, tc.want.To.Lat, result.To.Lat)
				assert.Equal(t, tc.want.To.Lng, result.To.Lng)
				assert.Equal(t, tc.want.UserID, result.UserID)
				assert.Greater(t, result.ID, int64(0))
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expected.Error(), err.Error())
			}
		})
	}
}

func Test_updateTravel(t *testing.T) {
	newTravel := func(id int64, fromLat, fromLng, toLat, toLng float64, status Status, userID int64) Travel {
		return Travel{
			ID:     id,
			Status: status,
			From: Point{
				Lat: fromLat,
				Lng: fromLng,
			},
			To: Point{
				Lat: toLat,
				Lng: toLng,
			},
			UserID: userID,
		}
	}

	tests := map[string]struct {
		db       repository
		trv      Travel
		expected error
	}{
		"successful travel update: change locations in pending": {
			db: newMockDBFromMap(map[int64]Travel{1: newTravel(1, -100, 70, 2, 20, StatusPending, 0)}),
			trv: Travel{
				ID: 1,
				From: Point{
					Lat: -120,
					Lng: 70,
				},
				To: Point{
					Lat: 22,
					Lng: 20,
				},
				Status: StatusPending,
			},
		},

		"successful travel update: change user id in pending": {
			db: newMockDBFromMap(map[int64]Travel{1: newTravel(1, -100, 70, 2, 20, StatusPending, 0)}),
			trv: Travel{
				ID: 1,
				From: Point{
					Lat: -100,
					Lng: 70,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
				Status: StatusPending,
				UserID: 1234,
			},
		},

		"failure travel update: change initial status without user on db travel": {
			db: newMockDBFromMap(map[int64]Travel{1: newTravel(1, -100, 70, 2, 20, StatusPending, 0)}),
			trv: Travel{
				ID: 1,
				From: Point{
					Lat: -100,
					Lng: 70,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
				Status: StatusInProcess,
			},
			expected: ErrInvalidUser,
		},

		"failure travel update: change locations in no pending status": {
			db: newMockDBFromMap(map[int64]Travel{1: newTravel(1, -100, 70, 2, 20, StatusInProcess, 0)}),
			trv: Travel{
				ID: 1,
				From: Point{
					Lat: 60,
					Lng: 2,
				},
				To: Point{
					Lat: -100,
					Lng: -33,
				},
				Status: StatusInProcess,
			},
			expected: ErrInvalidStatusToEditLocation,
		},

		"failure travel update: change user id in no pending status": {
			db: newMockDBFromMap(map[int64]Travel{1: newTravel(1, -100, 70, 2, 20, StatusInProcess, 12312312)}),
			trv: Travel{
				ID: 1,
				From: Point{
					Lat: -100,
					Lng: 70,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
				Status: StatusInProcess,
				UserID: 123,
			},
			expected: ErrInvalidUser,
		},

		"failure travel update: no user id in no pending status": {
			db: newMockDBFromMap(map[int64]Travel{1: newTravel(1, -100, 70, 2, 20, StatusInProcess, 12312312)}),
			trv: Travel{
				ID: 1,
				From: Point{
					Lat: -100,
					Lng: 70,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
				Status: StatusInProcess,
			},
			expected: ErrInvalidUser,
		},

		"failure travel update: no status": {
			db: newMockDBFromMap(map[int64]Travel{1: newTravel(1, -100, 70, 2, 20, StatusPending, 0)}),
			trv: Travel{
				ID: 1,
				From: Point{
					Lat: -100,
					Lng: 70,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
			},
			expected: ErrInvalidStatusToEdit,
		},

		"failure travel update: pending to ready": {
			db: newMockDBFromMap(map[int64]Travel{1: newTravel(1, -100, 70, 2, 20, StatusPending, 0)}),
			trv: Travel{
				ID: 1,
				From: Point{
					Lat: -100,
					Lng: 70,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
				Status: StatusReady,
				UserID: 1231,
			},
			expected: ErrInvalidStatusToEdit,
		},

		"failure travel update: in process to pending": {
			db: newMockDBFromMap(map[int64]Travel{1: newTravel(1, -100, 70, 2, 20, StatusInProcess, 1231)}),
			trv: Travel{
				ID: 1,
				From: Point{
					Lat: -100,
					Lng: 70,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
				Status: StatusPending,
				UserID: 1231,
			},
			expected: ErrInvalidStatusToEdit,
		},

		"db not found travel get": {
			db: newMockDB().onGet(22, ErrTravelNotFound),
			trv: Travel{
				ID: 22,
				From: Point{
					Lat: -100,
					Lng: 70,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
				Status: StatusPending,
				UserID: 1231,
			},
			expected: ErrNotFoundTravel,
		},

		"db failure travel update": {
			db: newMockDBFromMap(map[int64]Travel{22: newTravel(22, -100, 70, 2, 20, StatusPending, 0)}).
				onUpdate(22, errors.New("mocked db error")),
			trv: Travel{
				ID: 22,
				From: Point{
					Lat: -100,
					Lng: 70,
				},
				To: Point{
					Lat: 2,
					Lng: 20,
				},
				Status: StatusPending,
				UserID: 1234,
			},
			expected: ErrStorageUpdate,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			travelStorage := NewTravelStorage(tc.db)
			result, err := travelStorage.Update(context.Background(), tc.trv)

			if tc.expected == nil {
				assert.Nil(t, err)
				assert.NotNil(t, result)

				assert.Equal(t, tc.trv.Status, result.Status)
				assert.Equal(t, tc.trv.From.Lat, result.From.Lat)
				assert.Equal(t, tc.trv.From.Lng, result.From.Lng)
				assert.Equal(t, tc.trv.To.Lat, result.To.Lat)
				assert.Equal(t, tc.trv.To.Lng, result.To.Lng)
				assert.Equal(t, tc.trv.UserID, result.UserID)
				assert.Greater(t, result.ID, int64(0))
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expected.Error(), err.Error())
			}
		})
	}
}
