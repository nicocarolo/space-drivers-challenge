package travel

import (
	"context"
	"errors"
	"fmt"
)

type Status string

const (
	StatusPending   = "pending"
	StatusInProcess = "in_process"
	StatusReady     = "ready"
)

var taskFlow = []Status{StatusPending, StatusInProcess, StatusReady}

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
	ErrStorageSave                 = Error{code: "storage_failure", detail: "an error ocurred trying to save travel"}
	ErrStorageUpdate               = Error{code: "storage_failure", detail: "an error ocurred trying to update travel"}
	ErrStorageGet                  = Error{code: "storage_failure", detail: "an error ocurred trying to get travel"}
	ErrNotFoundTravel              = Error{code: "not_found_travel", detail: "not founded the travel to get"}
	ErrInvalidStatusToEditLocation = Error{code: "invalid_location_edit_status", detail: "travel status does not allow location change"}
	ErrInvalidStatusToEdit         = Error{code: "invalid_status", detail: "invalid received status"}
	ErrInvalidUser                 = Error{code: "invalid_user", detail: "invalid user while performing update"}
)

type Travel struct {
	ID     int64  `json:"id"`
	Status Status `json:"status"`
	From   Point  `json:"from" binding:"required"`
	To     Point  `json:"to" binding:"required"`
	UserID int64  `json:"user_id"`
}

type TravelStorage struct {
	repository repository
}

// NewTravelStorage will create and return a TravelStorage with the received repository
func NewTravelStorage(repository repository) TravelStorage {
	defaultUserStorage := TravelStorage{
		repository: repository,
	}

	return defaultUserStorage
}

// Get and return the travel with the received id from repository
func (travelStorage TravelStorage) Get(ctx context.Context, id int64) (Travel, error) {
	travel, err := travelStorage.repository.GetTravel(ctx, id)
	if err != nil {
		if errors.Is(err, ErrTravelNotFound) {
			return Travel{}, ErrNotFoundTravel
		}
		return Travel{}, ErrStorageGet
	}

	return travel, nil
}

// Save will store an User on repository and return it.
func (travelStorage TravelStorage) Save(ctx context.Context, travel Travel) (Travel, error) {
	travel.Status = StatusPending
	travel, err := travelStorage.repository.SaveTravel(ctx, travel)
	if err != nil {
		return Travel{}, ErrStorageSave
	}

	return travel, nil
}

// Update will update a stored travel on repository if the update satisfy validations and return it.
func (travelStorage TravelStorage) Update(ctx context.Context, newTravel Travel) (Travel, error) {
	travel, err := travelStorage.Get(ctx, newTravel.ID)
	if err != nil {
		return Travel{}, err
	}

	// validate there is no change in location if status on travel is not pending
	if (travel.From.Lat != newTravel.From.Lat || travel.From.Lng != newTravel.From.Lng ||
		travel.To.Lat != newTravel.To.Lat || travel.To.Lng != newTravel.To.Lng) && travel.Status != StatusPending {
		return Travel{}, ErrInvalidStatusToEditLocation
	}

	// validate status received is valid
	if newTravel.Status != StatusPending && newTravel.Status != StatusInProcess && newTravel.Status != StatusReady {
		return Travel{}, ErrInvalidStatusToEdit
	}

	// validate if status is not pending then the travel should have a user id
	if newTravel.UserID == 0 && travel.Status != StatusPending {
		return Travel{}, ErrInvalidUser
	}

	// validate if status received is not pending then the travel should have a user id
	if newTravel.UserID == 0 && newTravel.Status != StatusPending {
		return Travel{}, ErrInvalidUser
	}

	// validate if there is a change on the user id, when the travel already have a user, then the status received
	// it should be pending
	if newTravel.UserID != travel.UserID && travel.UserID != 0 && newTravel.Status != StatusPending {
		return Travel{}, ErrInvalidUser
	}

	currentlyStatus := findInSlice(taskFlow, Status(travel.Status))
	newStatus := findInSlice(taskFlow, Status(newTravel.Status))

	// validate new status, this can be only the same status or the next logical move
	// pending => in process
	// in process => ready
	if currentlyStatus != newStatus && currentlyStatus+1 != newStatus {
		return Travel{}, ErrInvalidStatusToEdit
	}

	travel.Status = newTravel.Status
	travel.UserID = newTravel.UserID
	travel.From = newTravel.From
	travel.To = newTravel.To

	err = travelStorage.repository.EditTravel(ctx, travel)
	if err != nil {
		return Travel{}, ErrStorageUpdate
	}

	return travel, nil
}

func findInSlice(s []Status, e Status) int {
	for i, a := range s {
		if a == e {
			return i
		}
	}
	return -1
}
