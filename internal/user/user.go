package user

import (
	"context"
	"errors"
	"fmt"
	"github.com/nicocarolo/space-drivers/internal/platform/jwt"
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
	ErrInvalidPasswordToSave  = Error{code: "invalid_password", detail: "cannot assign received password to user"}
	ErrInvalidPasswordToLogin = Error{code: "invalid_password", detail: "the password received to login is invalid"}
	ErrStorageSave            = Error{code: "storage_failure", detail: "an error ocurred trying to save user"}
	ErrStorageGet             = Error{code: "storage_failure", detail: "an error ocurred trying to get user"}
	ErrNotFoundUser           = Error{code: "not_found_user", detail: "not founded the user to get"}
	ErrInvalidRole            = Error{code: "invalid_role", detail: "the received role should be admin or driver"}
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
		passwordEncrypter: bcryptEncrypt{},
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
	pwd, err := userStorage.passwordEncrypter.Encrypt(user.Password)
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

// Login receive an email and password from User, search the user on db and compare the password.
// If the user exists and password is correct then return a generated jwt token.
func (userStorage UserStorage) Login(ctx context.Context, user User) (string, error) {
	userGet, err := userStorage.repository.GetUserByEmail(ctx, user.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrNotFoundUser
		}
		return "", ErrStorageGet
	}

	err = userStorage.passwordEncrypter.Compare(userGet.Password, user.Password)
	if err != nil {
		return "", ErrInvalidPasswordToLogin
	}

	token, err := jwt.GenerateToken(userGet.ID, userGet.Role)
	if err != nil {
		return "", err
	}

	return token, nil
}

type Search struct {
	status StatusSearch
	offset int64
	limit  int64
}

type StatusSearch string

const (
	StatusSearchBusy = "busy"
	StatusSearchFree = "free"
	StatusSearchNone = "none"
)

func WithStatus(status StatusSearch) SearchOption {
	return func(s *Search) {
		s.status = status
	}
}

func WithOffset(offset int64) SearchOption {
	return func(s *Search) {
		s.offset = offset
	}
}

func WithLimit(limit int64) SearchOption {
	return func(s *Search) {
		s.limit = limit
	}
}

type SearchOption func(ust *Search)

type Metadata struct {
	Total   int64
	Pending int64
}

// Search users on repository by status (currently only free drivers) or with pagination
func (userStorage UserStorage) Search(ctx context.Context, opt ...SearchOption) ([]SecuredUser, Metadata, error) {
	// default search options
	search := Search{
		status: StatusSearchNone,
		offset: 0,
		limit:  20,
	}

	// apply options
	for _, option := range opt {
		option(&search)
	}

	var users []User
	var err error
	var metadata Metadata
	// if none status, then search all user with pagination
	if search.status == StatusSearchNone {
		var totalCount int64
		users, totalCount, err = userStorage.repository.GetPaginate(ctx, search.limit, search.offset)
		metadata.Total = totalCount
		metadata.Pending = totalCount - search.limit - search.offset
		if metadata.Pending < 0 {
			metadata.Pending = 0
		}
	} else {
		// get free drivers
		users, err = userStorage.repository.GetFreeDrivers(ctx)
		metadata.Total = int64(len(users))
		metadata.Pending = 0
	}

	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, Metadata{}, ErrNotFoundUser
		}
		return nil, Metadata{}, ErrStorageGet
	}

	var secUsers []SecuredUser
	for _, u := range users {
		secUsers = append(secUsers, u.SecuredUser)
	}

	return secUsers, metadata, nil
}
