package travel

import (
	"context"
	"errors"
	"github.com/nicocarolo/space-drivers/internal/platform/code_error"
	"github.com/nicocarolo/space-drivers/internal/platform/jwt"
	"github.com/nicocarolo/space-drivers/internal/platform/log"
	"github.com/nicocarolo/space-drivers/internal/user"
)

type Status string

const (
	StatusPending   = "pending"
	StatusInProcess = "in_process"
	StatusReady     = "ready"
)

var travelFlow = []Status{StatusPending, StatusInProcess, StatusReady}

var (
	ErrStorageSave                 = code_error.Error{Code: "storage_failure", Detail: "an error ocurred trying to save travel"}
	ErrStorageUpdate               = code_error.Error{Code: "storage_failure", Detail: "an error ocurred trying to update travel"}
	ErrStorageGet                  = code_error.Error{Code: "storage_failure", Detail: "an error ocurred trying to get travel"}
	ErrNotFoundTravel              = code_error.Error{Code: "not_found_travel", Detail: "not founded the travel to get"}
	ErrInvalidStatusToEditLocation = code_error.Error{Code: "invalid_location_edit_status", Detail: "travel status does not allow location change"}
	ErrInvalidStatusToEdit         = code_error.Error{Code: "invalid_status", Detail: "invalid received status"}
	ErrInvalidUser                 = code_error.Error{Code: "invalid_user", Detail: "invalid user while performing update"}
	ErrInvalidUserClaims           = code_error.Error{Code: "invalid_user_access", Detail: "cannot identify user logged in"}
	ErrInvalidUserAccess           = code_error.Error{Code: "invalid_user_access", Detail: "the user logged in cannot perform this action, he is not the owner of the travel or it is not an admin"}
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
		log.Error(ctx, "there was an error while getting travel", log.Err(err))
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
		log.Error(ctx, "there was an error while saving travel", log.Err(err))
		return Travel{}, ErrStorageSave
	}

	return travel, nil
}

// Update will update a stored travel on repository if the update satisfy validations and return it.
func (travelStorage TravelStorage) Update(ctx context.Context, newTravel Travel) (Travel, error) {
	travel, err := travelStorage.Get(ctx, newTravel.ID)
	if err != nil {
		log.Error(ctx, "there was an error while getting travel on update", log.Int64("travel_id", travel.ID), log.Err(err))
		return Travel{}, err
	}

	// get user logged to check if he can change this travel
	userLogged, ok := ctx.Value("user_on_call").(jwt.Claims)
	if !ok {
		log.Info(ctx, "there was an error trying to access to user logged in claims",
			log.Int64("travel_user_id", travel.UserID),
			log.Int64("travel_id", travel.ID),
		)
		return Travel{}, ErrInvalidUserClaims
	}

	if err := validateTravelUpdate(ctx, travel, newTravel, userLogged); err != nil {
		return Travel{}, err
	}

	travel.Status = newTravel.Status
	travel.UserID = newTravel.UserID
	travel.From = newTravel.From
	travel.To = newTravel.To

	err = travelStorage.repository.EditTravel(ctx, travel)
	if err != nil {
		log.Error(ctx, "there was an error while updating travel", log.Int64("travel_id", travel.ID), log.Err(err))
		return Travel{}, ErrStorageUpdate
	}

	return travel, nil
}

func findStatusInFlow(e Status) int {
	for i, a := range travelFlow {
		if a == e {
			return i
		}
	}
	return -1
}

// validateTravelUpdate business validation on update travel
func validateTravelUpdate(ctx context.Context, travel Travel, changes Travel, userLogged jwt.Claims) error {
	isPending := travel.Status == StatusPending
	isChangeToPending := changes.Status == StatusPending

	changedLocation := travel.From.Lat != changes.From.Lat || travel.From.Lng != changes.From.Lng ||
		travel.To.Lat != changes.To.Lat || travel.To.Lng != changes.To.Lng

	changedUserID := changes.UserID != travel.UserID

	currentlyStatusIndex := findStatusInFlow(travel.Status)
	newStatusIndex := findStatusInFlow(changes.Status)

	changedStatus := newStatusIndex != currentlyStatusIndex

	// if the authenticated user is not the owner of the travel nor an admin then it cannot update the travel
	if travel.UserID != userLogged.UserID && userLogged.Role != user.RoleAdmin {
		log.Info(ctx, "there was an invalid check with user id on travel to update and user who is logged in",
			log.Int64("travel_id", travel.ID),
			log.Int64("travel_user_id", travel.UserID),
			log.Int64("logged_user_id", userLogged.UserID),
			log.String("logged_role", userLogged.Role),
		)
		return ErrInvalidUserAccess
	}

	// the user id assigned to the travel is changed but the role from the user authenticated is not admin
	if changedUserID && travel.UserID != 0 && userLogged.Role != user.RoleAdmin {
		log.Info(ctx, "there was an invalid check with user id on travel to update and user who is logged in: cannot change user id on travel with driver role",
			log.Int64("travel_id", travel.ID),
			log.Int64("travel_user_id", travel.UserID),
			log.Int64("logged_user_id", userLogged.UserID),
			log.String("logged_role", userLogged.Role),
		)
		return ErrInvalidUserAccess
	}

	// validate there is no change in location if status on travel is not pending
	if changedLocation && !isPending {
		log.Info(ctx, "invalid check on update travel: modifying locations when travel is not pending",
			log.Int64("travel_id", changes.ID),
			log.String("travel_status", string(travel.Status)))
		return ErrInvalidStatusToEditLocation
	}

	// validate status received is valid (findStatusInFlow return -1 when is invalid status = not find on travel flow)
	if newStatusIndex == -1 {
		log.Info(ctx, "invalid check on update travel: invalid status",
			log.Int64("travel_id", changes.ID),
			log.String("travel_status", string(changes.Status)))
		return ErrInvalidStatusToEdit
	}

	// validate if travel currently status is not pending then the travel change should have a user id
	if !isPending && changes.UserID == 0 {
		log.Info(ctx, "invalid check on update travel: no user id on update when is not in pending status",
			log.Int64("travel_id", changes.ID),
			log.String("travel_status", string(changes.Status)))
		return ErrInvalidUser
	}

	// validate if status received is not pending then the travel should have a user id
	if !isChangeToPending && changes.UserID == 0 {
		log.Info(ctx, "invalid check on update travel: no user id on update when change has no pending status",
			log.Int64("travel_id", changes.ID),
			log.String("travel_status", string(changes.Status)))
		return ErrInvalidUser
	}

	// validate if there is a change on the user id, when the travel already have a user, then the status received
	// it should be pending
	if changedUserID && travel.UserID != 0 && !isChangeToPending {
		log.Info(ctx, "invalid check on update travel: trying to change user when travel is not pending",
			log.Int64("travel_id", changes.ID),
			log.Int64("travel_user_id", changes.UserID),
			log.String("travel_status", string(changes.Status)))
		return ErrInvalidUser
	}

	// validate new status, this can be only the same status or the next logical move
	// pending => in process => ready
	if changedStatus && currentlyStatusIndex+1 != newStatusIndex {
		log.Info(ctx, "invalid check on update travel: invalid change of status",
			log.Int64("travel_id", changes.ID),
			log.String("travel_new_status", string(changes.Status)),
			log.String("travel_status", string(travel.Status)))
		return ErrInvalidStatusToEdit
	}

	return nil
}
