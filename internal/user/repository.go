package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nicocarolo/space-drivers/internal/platform/metrics"
	"strconv"
	"time"
)

const (
	dbUser     = "root"
	dbPassword = "root"
	dbname     = "space_drivers"

	timeMetricName   = "application.space.repository.time"
	entityMetricName = "user"
)

var ErrUserNotFound = errors.New("not founded user")

type repository interface {
	SaveUser(ctx context.Context, user User) (User, error)
	GetUser(ctx context.Context, id int64) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetFreeDrivers(ctx context.Context) ([]User, error)
	GetPaginate(ctx context.Context, limit, offset int64) ([]User, int64, error)
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

	trackTime := trackElapsed(ctx, entityMetricName, "insert")
	result, err := q.Exec(user.Email, user.Password, user.Role)
	trackTime(err == nil)
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

	trackTime := trackElapsed(ctx, entityMetricName, "select")
	newRecord := query.QueryRowContext(ctx, id)

	var user User
	err = newRecord.Scan(&user.ID, &user.Email, &user.Password, &user.Role)
	trackTime(err == nil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	return user, nil
}

func (sqlDb SqlRepository) GetPaginate(ctx context.Context, limit, offset int64) ([]User, int64, error) {
	queryStatement := fmt.Sprintf("SELECT id, role, email FROM users LIMIT %d, %d", limit, offset)
	if offset == 0 {
		queryStatement = fmt.Sprintf("SELECT id, role, email FROM users LIMIT %d", limit)
	}

	query, err := sqlDb.db.Prepare(queryStatement)
	if err != nil {
		return nil, 0, err
	}

	defer query.Close()

	trackTime := trackElapsed(ctx, entityMetricName, "select_paginate")
	rows, err := query.QueryContext(ctx)
	trackTime(err == nil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, ErrUserNotFound
		}
		return nil, 0, err

	}

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Role, &user.Email)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, 0, ErrUserNotFound
			}
			return nil, 0, err
		}

		users = append(users, user)
	}

	queryStatement = "SELECT COUNT(*) FROM users"

	trackTime = trackElapsed(ctx, entityMetricName, "select_count")
	query, err = sqlDb.db.Prepare(queryStatement)
	trackTime(err == nil)

	if err != nil {
		return nil, 0, err
	}

	defer query.Close()

	newRecord := query.QueryRowContext(ctx)

	var count int64
	err = newRecord.Scan(&count)

	return users, count, nil
}

func (sqlDb SqlRepository) GetFreeDrivers(ctx context.Context) ([]User, error) {
	queryStatement := fmt.Sprintf("SELECT id, role, email FROM users WHERE role = 'driver' AND id NOT IN " +
		"(select user_id from travels WHERE user_id IS NOT NULL AND (status = 'Pending' OR status = 'in_process'))")

	query, err := sqlDb.db.Prepare(queryStatement)
	if err != nil {
		return nil, err
	}

	defer query.Close()

	trackTime := trackElapsed(ctx, entityMetricName, "select_free")
	rows, err := query.QueryContext(ctx)
	trackTime(err == nil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err

	}

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Role, &user.Email)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrUserNotFound
			}
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

// GetUser will get a User who has the received id from table
func (sqlDb SqlRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	queryStatement := fmt.Sprintf("SELECT * FROM users WHERE email = ?")

	query, err := sqlDb.db.Prepare(queryStatement)
	if err != nil {
		return User{}, err
	}

	defer query.Close()

	trackTime := trackElapsed(ctx, entityMetricName, "select_by_email")
	newRecord := query.QueryRowContext(ctx, email)

	var user User
	err = newRecord.Scan(&user.ID, &user.Email, &user.Password, &user.Role)
	trackTime(err == nil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, err
	}

	return user, nil
}

func trackElapsed(ctx context.Context, entity, action string) func(success bool) {
	start := time.Now()
	return func(success bool) {
		metrics.Timing(ctx, timeMetricName, time.Since(start),
			[]string{
				"result", strconv.FormatBool(success),
				"action", action,
				"entity", entity})
	}
}
