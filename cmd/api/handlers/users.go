package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/nicocarolo/space-drivers/internal/user"
	"net/http"
	"strconv"
	"strings"
)

type UsersStorage interface {
	Get(ctx context.Context, id int64) (user.SecuredUser, error)
	Save(ctx context.Context, user user.User) (user.SecuredUser, error)
	Login(ctx context.Context, user user.User) (string, error)
}

type UserHandler struct {
	Users UsersStorage
}

// Get handler will parse received id as url param and get the user from storage
func (h UserHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, apiError{
			Code:        "invalid_request",
			Description: "the request has not a user id to get",
		})
		return
	}

	userResp, err := h.Users.Get(c, id)
	if err != nil {
		code, resp := mapUserError(err)
		c.JSON(code, resp)
		return
	}

	c.JSON(http.StatusOK, userResp)
}

// Create handler will parse received body and save it to storage
func (h UserHandler) Create(c *gin.Context) {
	var userToCreate user.User
	if err := c.ShouldBindJSON(&userToCreate); err != nil {
		apiErr := mapValidateError(err)
		c.JSON(http.StatusUnprocessableEntity, apiErr)
		return
	}

	createdUser, err := h.Users.Save(c, userToCreate)
	if err != nil {
		code, resp := mapUserError(err)
		c.JSON(code, resp)
		return
	}

	c.JSON(http.StatusCreated, createdUser)
}

type apiError struct {
	Code        string `json:"code,omitempty"`
	Description string `json:"description"`
}

func (e apiError) Error() string {
	return fmt.Sprintf("%s - %s", e.Code, e.Description)
}

// mapUserError received an error (preferentially a one received from storage) and return a http status code and
// an api error to use on the return value to the client
func mapUserError(err error) (int, error) {
	errToStatus := map[user.Error]int{
		user.ErrInvalidPasswordToSave: http.StatusBadRequest,
		user.ErrInvalidRole:           http.StatusBadRequest,
		user.ErrStorageSave:           http.StatusInternalServerError,
		user.ErrNotFoundUser:          http.StatusNotFound,
		user.ErrStorageGet:            http.StatusInternalServerError,
	}

	var userErr user.Error
	if errors.As(err, &userErr) {
		if code, ok := errToStatus[userErr]; ok {
			return code, apiError{
				Code:        userErr.Code(),
				Description: userErr.Detail(),
			}
		}
	}

	return http.StatusInternalServerError, apiError{
		Code:        "error",
		Description: err.Error(),
	}
}

// mapValidateError parse an error as it would be a validator package error and return an api error with
// fields that failed on validation
func mapValidateError(err error) apiError {
	validatorErr := validator.ValidationErrors{}
	if errors.As(err, &validatorErr) {
		var fields []string
		for _, fieldError := range validatorErr {
			fields = append(fields, fieldError.Field())
		}
		return apiError{
			Code:        "invalid_request",
			Description: fmt.Sprintf("there was an error with fields: %s", strings.ToLower(strings.Join(fields, ","))),
		}
	}

	return apiError{
		Code:        "invalid_request",
		Description: "the received request is invalid",
	}
}
