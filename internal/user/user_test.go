package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

type FailureEncrypter struct{}

func (f FailureEncrypter) Encrypt(pwd string) ([]byte, error) {
	return nil, fmt.Errorf("mocked password crypto error")
}

func (f FailureEncrypter) Compare(encrypted, pwd string) error {
	return fmt.Errorf("mocked password crypto error")
}

type NoEncrypter struct{}

func (n NoEncrypter) Encrypt(pwd string) ([]byte, error) {
	if strings.Contains(pwd, "error") {
		return nil, errors.New("mock encrypt error")
	}
	return []byte(pwd), nil
}

func (n NoEncrypter) Compare(encrypted, pwd string) error {
	if strings.Contains(pwd, "error") {
		return errors.New("mock encrypt error")
	}
	return nil
}

// mockDb a 'db' to use on UserStorage test with the capabilities to mock errors on create/get action
type mockDb struct {
	idCount int64
	users   map[int64]User

	saveError           map[string]error
	getError            map[int64]error
	getFreeDriversError error
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

func (db mockDb) GetUserByEmail(ctx context.Context, email string) (User, error) {
	for _, u := range db.users {
		if u.Email == email {
			return u, nil
		}
	}
	return User{}, ErrUserNotFound
}

func (db mockDb) GetFreeDrivers(ctx context.Context) ([]User, error) {
	if db.getFreeDriversError != nil {
		return nil, db.getFreeDriversError
	}
	return []User{
		User{
			SecuredUser: SecuredUser{
				ID:    1,
				Email: "an_email@hotmail.com",
				Role:  "driver",
			},
		},
		User{
			SecuredUser: SecuredUser{
				ID:    2,
				Email: "another_email@hotmail.com",
				Role:  "driver",
			},
		},
	}, nil
}

func (db mockDb) GetPaginate(ctx context.Context, limit, offset int64) ([]User, int64, error) {
	users := []User{
		User{
			SecuredUser: SecuredUser{
				ID:    1,
				Email: "an_email@hotmail.com",
				Role:  "driver",
			},
		},
		User{
			SecuredUser: SecuredUser{
				ID:    2,
				Email: "another_email@hotmail.com",
				Role:  "driver",
			},
		},
		User{
			SecuredUser: SecuredUser{
				ID:    3,
				Email: "another_email3@hotmail.com",
				Role:  "driver",
			},
		},
	}

	top := int64(len(users))
	if limit+offset < top {
		top = limit + offset
	}
	return users[offset:top], int64(len(users)), nil
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
			db:          newMockDB().onCreate("failure_email@hotmail.com", fmt.Errorf("mock db save error")),
			storageOpts: []UserStorageOption{WithPasswordEncrypter(FailureEncrypter{})},
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

func Test_loginUser(t *testing.T) {
	// config secret
	_ = os.Setenv("JWT_SECRET", "jdnfksdmfksd")

	dbWithUser := newMockDB()
	_, _ = dbWithUser.SaveUser(context.Background(), User{
		SecuredUser: SecuredUser{
			Email: "anEmail@asa.com",
			Role:  "admin",
		},
		Password: "a pass",
	})

	tests := map[string]struct {
		db        repository
		user      User
		encrypter PasswordEncrypter
		expected  error
	}{
		"successful user login": {
			db: dbWithUser,
			user: User{
				SecuredUser: SecuredUser{
					Email: "anEmail@asa.com",
				},
				Password: "a pass",
			},
			encrypter: NoEncrypter{},
		},

		"db failure user not found": {
			db: newMockDB(),
			user: User{
				SecuredUser: SecuredUser{
					Email: "nonexistemail@asa.com",
				},
				Password: "a pass",
			},
			expected: ErrNotFoundUser,
		},

		"db failure compare error": {
			db: dbWithUser,
			user: User{
				SecuredUser: SecuredUser{
					Email: "anEmail@asa.com",
				},
				Password: "a pass",
			},
			encrypter: FailureEncrypter{},
			expected:  ErrInvalidPasswordToLogin,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			userStorage := NewUserStorage(tc.db, WithPasswordEncrypter(tc.encrypter))
			result, err := userStorage.Login(context.Background(), tc.user)

			if tc.expected == nil {
				assert.Nil(t, err)
				assert.NotNil(t, result)

				assert.NotEmpty(t, result)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expected.Error(), err.Error())
			}
		})
	}
}

func Test_searchUser(t *testing.T) {
	tests := map[string]struct {
		db           repository
		opts         []SearchOption
		wantUsers    []SecuredUser
		wantMetadata Metadata
		expected     error
	}{
		"successful free drivers search": {
			db:   newMockDB(),
			opts: []SearchOption{WithStatus(StatusSearchFree)},
			wantUsers: []SecuredUser{
				{
					ID:    1,
					Email: "an_email@hotmail.com",
					Role:  "driver",
				},
				{
					ID:    2,
					Email: "another_email@hotmail.com",
					Role:  "driver",
				},
			},
			wantMetadata: Metadata{
				Total:   2,
				Pending: 0,
			},
		},

		"failure free drivers search: not found": {
			db:       newMockDB().onGetFreeDrivers(ErrUserNotFound),
			opts:     []SearchOption{WithStatus(StatusSearchFree)},
			expected: ErrNotFoundUser,
		},

		"failure free drivers search: storage error": {
			db:       newMockDB().onGetFreeDrivers(errors.New("mock st error")),
			opts:     []SearchOption{WithStatus(StatusSearchFree)},
			expected: ErrStorageGet,
		},

		"successful get driver paginate search": {
			db:   newMockDB(),
			opts: []SearchOption{},
			wantUsers: []SecuredUser{
				{
					ID:    1,
					Email: "an_email@hotmail.com",
					Role:  "driver",
				},
				{
					ID:    2,
					Email: "another_email@hotmail.com",
					Role:  "driver",
				},
				{
					ID:    3,
					Email: "another_email3@hotmail.com",
					Role:  "driver",
				},
			},
			wantMetadata: Metadata{
				Total:   3,
				Pending: 0,
			},
		},

		"successful get driver paginate search with limit": {
			db:   newMockDB(),
			opts: []SearchOption{WithLimit(2)},
			wantUsers: []SecuredUser{
				{
					ID:    1,
					Email: "an_email@hotmail.com",
					Role:  "driver",
				},
				{
					ID:    2,
					Email: "another_email@hotmail.com",
					Role:  "driver",
				},
			},
			wantMetadata: Metadata{
				Total:   3,
				Pending: 1,
			},
		},

		"successful get driver paginate search with offset": {
			db:   newMockDB(),
			opts: []SearchOption{WithOffset(1)},
			wantUsers: []SecuredUser{
				{
					ID:    2,
					Email: "another_email@hotmail.com",
					Role:  "driver",
				},
				{
					ID:    3,
					Email: "another_email3@hotmail.com",
					Role:  "driver",
				},
			},
			wantMetadata: Metadata{
				Total:   3,
				Pending: 0,
			},
		},

		"successful get driver paginate search with limit and offset": {
			db:   newMockDB(),
			opts: []SearchOption{WithLimit(1), WithOffset(1)},
			wantUsers: []SecuredUser{
				{
					ID:    2,
					Email: "another_email@hotmail.com",
					Role:  "driver",
				},
			},
			wantMetadata: Metadata{
				Total:   3,
				Pending: 1,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			userStorage := NewUserStorage(tc.db)
			result, meta, err := userStorage.Search(context.Background(), tc.opts...)

			if tc.expected == nil {
				assert.Nil(t, err)
				assert.NotNil(t, result)

				assert.Len(t, result, len(tc.wantUsers))
				assert.Equal(t, tc.wantMetadata.Total, meta.Total)
				assert.Equal(t, tc.wantMetadata.Pending, meta.Pending)
				for i, securedUser := range tc.wantUsers {
					assert.Equal(t, securedUser.ID, result[i].ID)
					assert.Equal(t, securedUser.Email, result[i].Email)
					assert.Equal(t, securedUser.Email, result[i].Email)
					assert.Equal(t, securedUser.Role, result[i].Role)
				}
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, tc.expected.Error(), err.Error())
			}
		})
	}
}
