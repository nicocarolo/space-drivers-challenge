package handlers

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/nicocarolo/space-drivers/internal/platform/code_error"
	"github.com/nicocarolo/space-drivers/internal/platform/log"
	"github.com/nicocarolo/space-drivers/internal/travel"
	"github.com/nicocarolo/space-drivers/internal/user"
	"net/http"
	"strconv"
)

type TravelStorage interface {
	Get(ctx context.Context, id int64) (travel.Travel, error)
	Save(ctx context.Context, travel travel.Travel) (travel.Travel, error)
	Update(ctx context.Context, travel travel.Travel) (travel.Travel, error)
}

type TravelHandler struct {
	Travels TravelStorage
	Users   UsersStorage
}

// Get handler will parse received id as url param and get the travel from storage
func (h TravelHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError{
			Code:        "invalid_request",
			Description: "the request has not a travel id to get",
		})
		return
	}

	travelResp, err := h.Travels.Get(c, id)
	if err != nil {
		code, resp := mapTravelError(err)
		c.JSON(code, resp)
		return
	}

	c.JSON(http.StatusOK, travelResp)
}

// Create handler will parse received body and save it to storage
func (h TravelHandler) Create(c *gin.Context) {
	var travelToCreate travel.Travel
	if err := c.ShouldBindJSON(&travelToCreate); err != nil {
		apiErr := mapValidateError(err)
		c.JSON(http.StatusUnprocessableEntity, apiErr)
		return
	}

	createdTravel, err := h.Travels.Save(c, travelToCreate)
	if err != nil {
		code, resp := mapTravelError(err)
		c.JSON(code, resp)
		return
	}

	c.JSON(http.StatusCreated, createdTravel)
}

// Edit handler will parse received body and id and edit travel in to storage
func (h TravelHandler) Edit(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		log.Error(c, "there was an error getting id from request on edit travel", log.Err(err))
		c.JSON(http.StatusBadRequest, apiError{
			Code:        "invalid_request",
			Description: "the request has not a travel id to update",
		})
		return
	}

	var travelToUpdate travel.Travel
	if err := c.ShouldBindJSON(&travelToUpdate); err != nil {
		log.Error(c, "there was an error parsing travel edit request", log.Err(err))
		apiErr := mapValidateError(err)
		c.JSON(http.StatusUnprocessableEntity, apiErr)
		return
	}

	travelToUpdate.ID = id

	if travelToUpdate.UserID != 0 {
		_, err := h.Users.Get(c, travelToUpdate.UserID)
		if err != nil && errors.Is(err, user.ErrNotFoundUser) {
			c.JSON(http.StatusBadRequest, apiError{
				Code:        "invalid_travel_user",
				Description: "the user received was not found",
			})
			return
		}
	}

	createdTravel, err := h.Travels.Update(c, travelToUpdate)
	if err != nil {
		code, resp := mapTravelError(err)
		c.JSON(code, resp)
		return
	}

	c.JSON(http.StatusOK, createdTravel)
}

func mapTravelError(err error) (int, error) {
	errToStatus := map[code_error.Error]int{
		travel.ErrStorageSave:                 http.StatusInternalServerError,
		travel.ErrStorageUpdate:               http.StatusInternalServerError,
		travel.ErrStorageGet:                  http.StatusInternalServerError,
		travel.ErrNotFoundTravel:              http.StatusNotFound,
		travel.ErrInvalidStatusToEditLocation: http.StatusBadRequest,
		travel.ErrInvalidStatusToEdit:         http.StatusBadRequest,
		travel.ErrInvalidUser:                 http.StatusBadRequest,
		travel.ErrInvalidUserClaims:           http.StatusUnauthorized,
		travel.ErrInvalidUserAccess:           http.StatusUnauthorized,
	}

	var travelErr code_error.Error
	if errors.As(err, &travelErr) {
		if code, ok := errToStatus[travelErr]; ok {
			return code, apiError{
				Code:        travelErr.GetCode(),
				Description: travelErr.GetDetail(),
			}
		}
	}

	return http.StatusInternalServerError, apiError{
		Code:        "error",
		Description: err.Error(),
	}
}
