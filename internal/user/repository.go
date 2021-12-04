package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

const (
	dbUser     = "root"
	dbPassword = "root"
	dbname     = "space_drivers"
)

var ErrUserNotFound = errors.New("not founded user")

type repository interface {
	SaveUser(ctx context.Context, user User) (User, error)
	GetUser(ctx context.Context, id int64) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
}

// SqlRepository sql client wrapper for user model
type SqlRepository struct {
	db *sql.DB
}

// NewRepository creates and return an SqlRepository
func NewRepository() (SqlRepository, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@/%s", dbUser, dbPassword, dbname))
	if err != nil {
		return SqlRepository{}, err
	}

	return SqlRepository{
		db: db,
	}, nil
}

// SaveUser will store a User on sql table
func (sqlDb SqlRepository) SaveUser(ctx context.Context, user User) (User, error) {
	q, err := sqlDb.db.Prepare("INSERT INTO users(email, password, role) VALUES(?, ?, ?)")
	if err != nil {
		return User{}, err
	}
	result, err := q.Exec(user.Email, user.Password, user.Role)
	if err != nil {
		return User{}, err
	}

	defer q.Close()

	user.ID, err = result.LastInsertId()
	if err != nil {
		return User{}, err
	}

	return user, nil
}

// GetUser will get a User who has the received id from table
func (sqlDb SqlRepository) GetUser(ctx context.Context, id int64) (User, error) {
	queryStatement := fmt.Sprintf("SELECT * FROM users WHERE id = ?")

	query, err := sqlDb.db.Prepare(queryStatement)
	if err != nil {
		return User{}, err
	}

	defer query.Close()

	newRecord := query.QueryRowContext(ctx, id)

	var user User
	err = newRecord.Scan(&user.ID, &user.Email, &user.Password, &user.Role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	return user, nil
}

// GetUser will get a User who has the received id from table
func (sqlDb SqlRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	queryStatement := fmt.Sprintf("SELECT * FROM users WHERE email = ?")

	query, err := sqlDb.db.Prepare(queryStatement)
	if err != nil {
		return User{}, err
	}

	defer query.Close()

	newRecord := query.QueryRowContext(ctx, email)

	var user User
	err = newRecord.Scan(&user.ID, &user.Email, &user.Password, &user.Role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	return user, nil
}
