package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

// mockDb a 'db' to use on UserStorage test with the capabilities to mock errors on create/get action
type mockDb struct {
	idCount int64
	users   map[int64]User

	saveError map[string]error
	getError  map[int64]error
}

func (db *mockDb) onCreate(email string, err error) *mockDb {
	db.saveError[email] = err
	return db
}

func (db *mockDb) onGet(id int64, err error) *mockDb {
	db.getError[id] = err
	return db
}

func (db *mockDb) SaveUser(ctx context.Context, user User) (User, error) {
	if err, ok := db.saveError[user.Email]; ok {
		return User{}, err
	}

	user.ID = db.idCount
	db.users[user.ID] = user

	db.idCount++

	return user, nil
}

func (db mockDb) GetUser(ctx context.Context, id int64) (User, error) {
	if err, ok := db.getError[id]; ok {
		return User{}, err
	}

	user, exist := db.users[id]
	if !exist {
		return User{}, fmt.Errorf("not found user")
	}

	return user, nil
}

func newMockDB() *mockDb {
	return &mockDb{
		idCount: 1,
		users:   make(map[int64]User),

		saveError: make(map[string]error),
		getError:  make(map[int64]error),
	}
}

func Test_createUser(t *testing.T) {
	tests := map[string]struct {
		db          repository
		storageOpts []UserStorageOption
		us          User
		want        SecuredUser
		expected    error
	}{
		"successful user save": {
			db: newMockDB(),
			us: User{
				SecuredUser: SecuredUser{
					Email: "an_email@hotmail.com",
					Role:  "admin",
				},
				Password: "a_pass",
			},
			want: SecuredUser{
				Email: "an_email@hotmail.com",
				Role:  "admin",
			},
		},

		"db failure on user save": {
			db: newMockDB().onCreate("failure_email@hotmail.com", fmt.Errorf("mock db save error")),
			us: User{
				SecuredUser: SecuredUser{
					Email: "failure_email@hotmail.com",
					Role:  "admin",
				},
				Password: "a_pass",
			},
			expected: ErrStorageSave,
		},

		"invalid role failure on user save": {
			db: newMockDB(),
			us: User{
				SecuredUser: SecuredUser{
					Email: "failure_email@hotmail.com",
					Role:  "an invalid role",
				},
				Password: "a_pass",
			},
			expected: ErrInvalidRole,
		},

		"password failure on user save": {
			db: newMockDB().onCreate("failure_email@hotmail.com", fmt.Errorf("mock db save error")),
			storageOpts: []UserStorageOption{WithPasswordEncrypter(func(pwd string) ([]byte, error) {
				return nil, fmt.Errorf("mocked password crypto error")
			})},
			us: User{
				SecuredUser: SecuredUser{
					Email: "failure_email@hotmail.com",
					Role:  "admin",
				},
				Password: "",
			},
			expected: ErrInvalidPasswordToSave,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			userStorage := NewUserStorage(tc.db, tc.storageOpts...)
			result, err := userStorage.Save(context.Background(), tc.us)

			if tc.expected == nil {
				assert.Nil(t, err)
				assert.NotNil(t, result)

				assert.Equal(t, tc.want.Role, result.Role)
				assert.Equal(t, tc.want.Email, result.Email)
				assert.Greater(t, result.ID, int64(0))
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expected.Error(), err.Error())
			}
		})
	}
}

func Test_getUser(t *testing.T) {
	dbWithUser := newMockDB()
	createdUser, _ := dbWithUser.SaveUser(context.Background(), User{
		SecuredUser: SecuredUser{
			Email: "anEmail@asa.com",
			Role:  "admin",
		},
		Password: "a pass",
	})

	tests := map[string]struct {
		db       repository
		idToGet  int64
		want     SecuredUser
		expected error
	}{
		"successful user get": {
			db:      dbWithUser,
			idToGet: createdUser.ID,
			want: SecuredUser{
				ID:    createdUser.ID,
				Email: "anEmail@asa.com",
				Role:  "admin",
			},
		},

		"db failure user not found": {
			db:       newMockDB().onGet(22, ErrUserNotFound),
			idToGet:  22,
			expected: ErrNotFoundUser,
		},

		"db failure user get": {
			db:       newMockDB().onGet(22, errors.New("mocked get error")),
			idToGet:  22,
			expected: ErrStorageGet,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			userStorage := NewUserStorage(tc.db)
			result, err := userStorage.Get(context.Background(), tc.idToGet)

			if tc.expected == nil {
				assert.Nil(t, err)
				assert.NotNil(t, result)

				assert.Equal(t, tc.want.Role, result.Role)
				assert.Equal(t, tc.want.Email, result.Email)
				assert.Greater(t, result.ID, int64(0))
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expected.Error(), err.Error())
			}
		})
	}
}
