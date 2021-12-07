package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/nicocarolo/space-drivers/internal/platform/code_error"
	"github.com/nicocarolo/space-drivers/internal/user"
	"net/http"
	"strconv"
	"strings"
)

type UsersStorage interface {
	Get(ctx context.Context, id int64) (user.SecuredUser, error)
	Save(ctx context.Context, user user.User) (user.SecuredUser, error)
	Login(ctx context.Context, user user.User) (string, error)
	Search(ctx context.Context, opt ...user.SearchOption) ([]user.SecuredUser, user.Metadata, error)
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

// GetDrivers get driver by status, or pagination
// ?status={status}&limit={pageNumber}&offset={pageSize}
func (h UserHandler) GetDrivers(c *gin.Context) {
	status := c.Query("status")
	limit := c.Query("limit")
	offset := c.Query("offset")

	var searchOptions []user.SearchOption
	// validate status
	if status != "" /* && status != user.StatusSearchBusy */ && status != user.StatusSearchFree {
		// currently only free drivers search available
		c.JSON(http.StatusBadRequest, apiError{
			Code:        "invalid_request",
			Description: "invalid search status received",
		})
		return
	}

	// if status received
	if status != "" {
		// cannot receive limit and offset with status search
		if limit != "" || offset != "" {
			c.JSON(http.StatusBadRequest, apiError{
				Code:        "invalid_request",
				Description: "search free driver do not accept limit or offset param",
			})
			return
		}
		searchOptions = append(searchOptions, user.WithStatus(user.StatusSearch(status)))
	}

	// parse limit if it was received
	if limit != "" {
		limitNmbr, err := strconv.ParseInt(limit, 10, 64)
		if err != nil || limitNmbr == 0 {
			c.JSON(http.StatusBadRequest, apiError{
				Code:        "invalid_request",
				Description: "invalid search limit received",
			})
			return
		}
		searchOptions = append(searchOptions, user.WithLimit(limitNmbr))
	}

	// parse offset if it was received
	if offset != "" {
		offsetNmbr, err := strconv.ParseInt(offset, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, apiError{
				Code:        "invalid_request",
				Description: "invalid search offset received",
			})
			return
		}
		searchOptions = append(searchOptions, user.WithOffset(offsetNmbr))
	}

	userResp, meta, err := h.Users.Search(c, searchOptions...)
	if err != nil {
		code, resp := mapUserError(err)
		c.JSON(code, resp)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"total":   meta.Total,
		"pending": meta.Pending,
		"result":  userResp,
	})
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
	errToStatus := map[code_error.Error]int{
		user.ErrInvalidPasswordToSave: http.StatusBadRequest,
		user.ErrInvalidRole:           http.StatusBadRequest,
		user.ErrStorageSave:           http.StatusInternalServerError,
		user.ErrNotFoundUser:          http.StatusNotFound,
		user.ErrStorageGet:            http.StatusInternalServerError,
	}

	var userErr code_error.Error
	if errors.As(err, &userErr) {
		if code, ok := errToStatus[userErr]; ok {
			return code, apiError{
				Code:        userErr.GetCode(),
				Description: userErr.GetDetail(),
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
