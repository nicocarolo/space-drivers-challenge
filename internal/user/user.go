package user

import (
	"context"
	"errors"
	"fmt"
)

const (
	RoleAdmin  = "admin"
	RoleDriver = "driver"
)

type Error struct {
	code   string
	detail string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s - %s", e.code, e.detail)
}

func (e Error) Code() string {
	return e.code
}

func (e Error) Detail() string {
	return e.detail
}

var (
	ErrInvalidPasswordToSave = Error{code: "invalid_password", detail: "cannot assign received password to user"}
	ErrStorageSave           = Error{code: "storage_failure", detail: "an error ocurred trying to save user"}
	ErrStorageGet            = Error{code: "storage_failure", detail: "an error ocurred trying to get user"}
	ErrNotFoundUser          = Error{code: "not_found_user", detail: "not founded the user to get"}
	ErrInvalidRole           = Error{code: "invalid_role", detail: "the received role should be admin or driver"}
)

// WithPasswordEncrypter will change the algorithm to encrypt password with the received
func WithPasswordEncrypter(enc PasswordEncrypter) UserStorageOption {
	return func(ust *UserStorage) {
		ust.passwordEncrypter = enc
	}
}

type SecuredUser struct {
	ID    int64  `json:"id"`
	Email string `json:"email" binding:"required"`
	Role  string `json:"role" binding:"required"`
}

type User struct {
	SecuredUser
	Password string `json:"password" binding:"required"`
}

type UserStorage struct {
	repository        repository
	passwordEncrypter PasswordEncrypter
}

// UserStorageOption type to change UserStorage configuration
type UserStorageOption func(ust *UserStorage)

// NewUserStorage will create and return a UserStorage with the received repository and applying the options
// Default options are:
// 	- bcryptEncrypter to encrypt password
func NewUserStorage(repository repository, opts ...UserStorageOption) UserStorage {
	defaultUserStorage := UserStorage{
		repository:        repository,
		passwordEncrypter: bcryptEncrypter(),
	}

	for _, opt := range opts {
		opt(&defaultUserStorage)
	}

	return defaultUserStorage
}

// Get and return the User from repository with the received id
func (userStorage UserStorage) Get(ctx context.Context, id int64) (SecuredUser, error) {
	user, err := userStorage.repository.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return SecuredUser{}, ErrNotFoundUser
		}
		return SecuredUser{}, ErrStorageGet
	}

	return SecuredUser{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}, nil
}

// Save will store an User on repository and return it.
// The password received is encrypted with passwordEncrypter on UserStorage, and the roles accepted are
// 'admin' or 'driver's
func (userStorage UserStorage) Save(ctx context.Context, user User) (SecuredUser, error) {
	pwd, err := userStorage.passwordEncrypter(user.Password)
	if err != nil {
		return SecuredUser{}, ErrInvalidPasswordToSave
	}

	user.Password = string(pwd)

	if user.Role != RoleDriver && user.Role != RoleAdmin {
		return SecuredUser{}, ErrInvalidRole
	}

	user, err = userStorage.repository.SaveUser(ctx, user)
	if err != nil {
		return SecuredUser{}, ErrStorageSave
	}

	return SecuredUser{
		ID:    user.ID,
		Email: user.Email,
		Role:  user.Role,
	}, nil
}
