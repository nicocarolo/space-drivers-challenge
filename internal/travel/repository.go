package travel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"reflect"
)

const (
	dbUser     = "root"
	dbPassword = "root"
	dbname     = "space_drivers"
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
// CREATE TABLE travel
// (
//		id int PRIMARY KEY NOT NULL AUTO_INCREMENT,
//		user_id int,
//		`from` varchar(50) NOT NULL,
//		`to` varchar(50) NOT NULL,
//		status varchar(15) NOT NULL
// );
// CREATE UNIQUE INDEX travel_id_uindex ON travel (id);
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
func (sqlDb SqlRepository) SaveTravel(ctx context.Context, travel Travel) (Travel, error) {
	q, err := sqlDb.db.Prepare("INSERT INTO travels(status, `from`, `to`, user_id) VALUES(?, ?, ?, ?)")
	if err != nil {
		return Travel{}, err
	}

	var userID interface{}
	if travel.UserID != 0 {
		userID = travel.UserID
	}

	result, err := q.Exec(travel.Status, travel.From.String(), travel.To.String(), userID)
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

	result, err := q.Exec(travel.Status, travel.From.String(), travel.To.String(), travel.UserID, travel.ID)
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

	newRecord := query.QueryRowContext(ctx, id)

	var travel Travel
	var from string
	var to string
	var userID sql.NullInt64
	err = newRecord.Scan(&travel.ID, &travel.Status, &from, &to, &userID)
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

// NullInt64 is an alias for sql.NullInt64 data type
type NullInt64 sql.NullInt64

// Scan implements the Scanner interface for NullInt64
func (ni *NullInt64) Scan(value interface{}) error {
	var i sql.NullInt64
	if err := i.Scan(value); err != nil {
		return err
	}

	// if nil then make Valid false
	if reflect.TypeOf(value) == nil {
		*ni = NullInt64{i.Int64, false}
	} else {
		*ni = NullInt64{i.Int64, true}
	}
	return nil
}