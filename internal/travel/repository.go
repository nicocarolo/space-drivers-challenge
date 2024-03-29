package travel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nicocarolo/space-drivers/internal/platform/metrics"
	"os"
	"strconv"
	"time"
)

const (
	dbnameDefault = "space_drivers"

	timeMetricName   = "application.space.repository.time"
	entityMetricName = "travel"
)

var (
	ErrTravelNotFound         = errors.New("not founded travel")
	ErrTravelNotFoundOnUpdate = errors.New("not founded travel on update")
	ErrInvalidFromLocation    = errors.New("invalid 'from' location")
	ErrInvalidToLocation      = errors.New("invalid 'to' location")
)

type repository interface {
	SaveTravel(ctx context.Context, travel Travel) (Travel, error)
	EditTravel(ctx context.Context, travel Travel) error
	GetTravel(ctx context.Context, id int64) (Travel, error)
}

// SqlRepository sql client wrapper for user model
type SqlRepository struct {
	db *sql.DB
}

// NewRepository creates and return an SqlRepository
func NewRepository() (SqlRepository, error) {
	dbname := os.Getenv("DB_NAME")
	dbuser := os.Getenv("DB_USER")
	dbpass := os.Getenv("DB_PASSWORD")
	dbimage := os.Getenv("DB_IMAGE_NAME")
	scope := os.Getenv("SCOPE")

	if dbname == "" {
		dbname = dbnameDefault
	}
	if dbuser == "" || dbpass == "" || dbimage == "" {
		return SqlRepository{}, fmt.Errorf("cannot initialize user repository: the following settings " +
			"(DB_USER, DB_PASSWORD, DB_IMAGE_NAME) are invalid")
	}

	dataSourceConnection := fmt.Sprintf("%s:%s@/%s", dbuser, dbpass, dbname)
	if scope != "" {
		dataSourceConnection = fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", dbuser, dbpass, dbimage, dbname)
	}
	db, err := sql.Open("mysql", dataSourceConnection)
	if err != nil {
		return SqlRepository{}, err
	}

	return SqlRepository{
		db: db,
	}, nil
}

// SaveUser will store a User on sql table
func (sqlDb SqlRepository) SaveTravel(ctx context.Context, travel Travel) (Travel, error) {
	q, err := sqlDb.db.Prepare("INSERT INTO travels(status, `from`, `to`, user_id) VALUES(?, ?, ?, ?)")
	if err != nil {
		return Travel{}, err
	}

	var userID interface{}
	if travel.UserID != 0 {
		userID = travel.UserID
	}

	trackTime := trackElapsed(ctx, entityMetricName, "insert")
	result, err := q.Exec(travel.Status, travel.From.String(), travel.To.String(), userID)
	trackTime(err == nil)
	if err != nil {
		return Travel{}, err
	}

	travel.ID, err = result.LastInsertId()
	if err != nil {
		return Travel{}, err
	}

	return travel, nil
}

// SaveUser will store a User on sql table
func (sqlDb SqlRepository) EditTravel(ctx context.Context, travel Travel) error {
	q, err := sqlDb.db.Prepare("UPDATE travels SET status = ?, `from` = ?, `to` = ?, user_id = ? WHERE id = ?")
	if err != nil {
		return err
	}

	trackTime := trackElapsed(ctx, entityMetricName, "update")
	result, err := q.Exec(travel.Status, travel.From.String(), travel.To.String(), travel.UserID, travel.ID)
	trackTime(err == nil)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected != 1 {
		return ErrTravelNotFoundOnUpdate
	}

	return nil
}

// GetUser will get a User who has the received id from table
func (sqlDb SqlRepository) GetTravel(ctx context.Context, id int64) (Travel, error) {
	queryStatement := fmt.Sprintf("SELECT id, status, `from`, `to`, user_id FROM travels WHERE id = ?")

	query, err := sqlDb.db.Prepare(queryStatement)
	if err != nil {
		return Travel{}, err
	}

	defer query.Close()

	trackTime := trackElapsed(ctx, entityMetricName, "select")
	newRecord := query.QueryRowContext(ctx, id)

	var travel Travel
	var from string
	var to string
	var userID sql.NullInt64
	err = newRecord.Scan(&travel.ID, &travel.Status, &from, &to, &userID)
	trackTime(err == nil)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Travel{}, ErrTravelNotFound
		}
		return Travel{}, err
	}

	if userID.Valid {
		travel.UserID = userID.Int64
	}

	err = travel.From.FromString(from)
	if err != nil {
		return Travel{}, ErrInvalidFromLocation
	}

	err = travel.To.FromString(to)
	if err != nil {
		return Travel{}, ErrInvalidToLocation
	}

	return travel, nil
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
