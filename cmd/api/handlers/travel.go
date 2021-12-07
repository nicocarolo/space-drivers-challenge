package handlers

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/nicocarolo/space-drivers/internal/platform/jwt"
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
		c.JSON(http.StatusBadRequest, apiError{
			Code:        "invalid_request",
			Description: "the request has not a travel id to update",
		})
		return
	}

	var travelToUpdate travel.Travel
	if err := c.ShouldBindJSON(&travelToUpdate); err != nil {
		apiErr := mapValidateError(err)
		c.JSON(http.StatusUnprocessableEntity, apiErr)
		return
	}

	travelToUpdate.ID = id

	userCall, exist := c.Get("user_on_call")
	if !exist {
		c.JSON(http.StatusUnauthorized, apiError{
			Code:        "invalid_request_user",
			Description: "cannot get user login",
		})
		return
	}
	userClaims := userCall.(jwt.Claims)

	if travelToUpdate.UserID != 0 {
		_, err := h.Users.Get(c, travelToUpdate.UserID)
		if err != nil && errors.Is(err, user.ErrNotFoundUser) {
			c.JSON(http.StatusBadRequest, apiError{
				Code:        "invalid_travel_user",
				Description: "the user received was not found",
			})
			return
		}

		existedTravel, err := h.Travels.Get(c, travelToUpdate.ID)
		if err != nil {
			code, resp := mapTravelError(err)
			c.JSON(code, resp)
			return
		}

		// if the user who is logged is not the owner of the travel, and it is not an admin then
		// it cannot update travel
		if existedTravel.UserID != userClaims.UserID && userClaims.Role != user.RoleAdmin {
			c.JSON(http.StatusUnauthorized, apiError{
				Code:        "invalid_request_user",
				Description: "the user who is logged is not the owner of the travel and it is not an admin",
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
	errToStatus := map[travel.Error]int{
		travel.ErrStorageSave:                 http.StatusInternalServerError,
		travel.ErrStorageUpdate:               http.StatusInternalServerError,
		travel.ErrStorageGet:                  http.StatusInternalServerError,
		travel.ErrNotFoundTravel:              http.StatusNotFound,
		travel.ErrInvalidStatusToEditLocation: http.StatusBadRequest,
		travel.ErrInvalidStatusToEdit:         http.StatusBadRequest,
		travel.ErrInvalidUser:                 http.StatusBadRequest,
	}

	var travelErr travel.Error
	if errors.As(err, &travelErr) {
		if code, ok := errToStatus[travelErr]; ok {
			return code, apiError{
				Code:        travelErr.Code(),
				Description: travelErr.Detail(),
			}
		}
	}

	return http.StatusInternalServerError, apiError{
		Code:        "error",
		Description: err.Error(),
	}
}
