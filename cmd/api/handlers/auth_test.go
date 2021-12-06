package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/nicocarolo/space-drivers/internal/user"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type NoEncrypter struct{}

func (n NoEncrypter) Encrypt(pwd string) ([]byte, error) {
	if strings.Contains(pwd, "error") {
		return nil, errors.New("mock encrypt error")
	}
	return []byte(pwd), nil
}

func (n NoEncrypter) Compare(encrypted, pwd string) error {
	if strings.Contains(pwd, "error") {
		return errors.New("mock encrypt error")
	}
	return nil
}

func Test_LoginUser(t *testing.T) {
	// config secret
	_ = os.Setenv("JWT_SECRET", "jdnfksdmfksd")

	userDB := newMockDB()
	userDB.SaveUser(context.Background(), user.User{
		SecuredUser: user.SecuredUser{
			Email: "an_email@",
			Role:  "admin",
		},
		Password: "1234",
	})
	testscases := map[string]struct {
		body           map[string]interface{}
		wantError      error
		statusExpected int
	}{
		"successful login": {
			body: map[string]interface{}{
				"email":    "an_email@",
				"password": "1234",
			},
			statusExpected: http.StatusOK,
		},

		"failure login due to invalid request: no email ": {
			body: map[string]interface{}{
				"password": "12313",
			},
			wantError:      errors.New("invalid_request - there was an error with fields: email"),
			statusExpected: http.StatusUnprocessableEntity,
		},

		"failure login due to invalid request: no password ": {
			body: map[string]interface{}{
				"email": "an_email@",
			},
			wantError:      errors.New("invalid_request - there was an error with fields: password"),
			statusExpected: http.StatusUnprocessableEntity,
		},

		"failure login due to encrypter error: no password": {
			body: map[string]interface{}{
				"email":    "an_email@",
				"password": "error",
			},
			wantError:      errors.New("invalid_password - the password received to login is invalid"),
			statusExpected: http.StatusBadRequest,
		},

		"failure login due to storage error: user not found": {
			body: map[string]interface{}{
				"email":    "anemail@",
				"password": "error",
			},
			wantError:      errors.New("not_found_user - not founded the user to get"),
			statusExpected: http.StatusNotFound,
		},
	}

	for name, tc := range testscases {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = &http.Request{
				Header: make(http.Header),
			}

			err := mockJson(c, http.MethodPost, tc.body)
			assert.Nil(t, err)

			handler := AuthHandler{
				Users: user.NewUserStorage(userDB, user.WithPasswordEncrypter(NoEncrypter{})),
			}
			handler.Login(c)

			assert.Equal(t, tc.statusExpected, w.Code)

			if tc.wantError != nil {
				var apiErr apiError
				err = json.Unmarshal(w.Body.Bytes(), &apiErr)
				assert.Nil(t, err)

				assert.Equal(t, tc.wantError.Error(), apiErr.Error())
			} else {
				var resp map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &resp)
				assert.Nil(t, err)

				assert.NotEmpty(t, resp["token"])
			}
		})
	}
}
