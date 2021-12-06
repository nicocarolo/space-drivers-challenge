package handlers

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nicocarolo/space-drivers/internal/platform/jwt"
	"github.com/nicocarolo/space-drivers/internal/platform/log"
	"github.com/nicocarolo/space-drivers/internal/user"
	"net/http"
)

type Authenticate interface {
	Login(ctx context.Context, user user.User) (string, error)
}

type AuthHandler struct {
	Users UsersStorage
}

// Login handler will receive an email and password and login a user returning a token to authenticate on future
// requests
func (h AuthHandler) Login(c *gin.Context) {
	type loginRequest struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var loginReq loginRequest
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		apiErr := mapValidateError(err)
		c.JSON(http.StatusUnprocessableEntity, apiErr)
		return
	}

	userToLogin := user.User{
		SecuredUser: user.SecuredUser{
			Email: loginReq.Email,
		},
		Password: loginReq.Password,
	}
	token, err := h.Users.Login(c, userToLogin)
	if err != nil {
		code, resp := mapAuthError(err)
		c.JSON(code, resp)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"token": token,
	})
}

func mapAuthError(err error) (int, error) {
	errToStatus := map[user.Error]int{
		user.ErrNotFoundUser:           http.StatusNotFound,
		user.ErrInvalidPasswordToLogin: http.StatusBadRequest,
		user.ErrStorageGet:             http.StatusInternalServerError,
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

// AuthenticateRequest authenticate the received request with the jwt token on Bearer header.
// The token is validated and if it is ok, the user on it is stored on context.
func AuthenticateRequest() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		const BearerSchema string = "Bearer "
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
				Code:        "authorization_token_missing",
				Description: "it was not received the authorization header with token",
			})
			return
		}
		tokenString := authHeader[len(BearerSchema):]

		token, err := jwt.ValidateToken(tokenString)
		if err != nil {
			log.Error(ctx, "there was an error validating token on authenticate request", log.Err(err))
			if errors.Is(err, jwt.ErrTokenExpired) {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
					Code:        "expired_token",
					Description: err.Error(),
				})
				return
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
				Code:        "invalid_token",
				Description: err.Error(),
			})
			return
		}

		claims, err := jwt.GetClaims(token)
		if err != nil {
			log.Error(ctx, "there was an error getting claims from token on authenticate request", log.Err(err))
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
				Code:        "invalid_token_data",
				Description: err.Error(),
			})
			return
		}

		ctx.Set("user_on_call", claims)
	}
}

// rule model to perform role based access control
type rule struct {
	url    string
	method string
	role   string
}

func newRule(url, method, role string) rule {
	return rule{
		url:    url,
		method: method,
		role:   role,
	}
}

type Ruler interface {
	CanAccess(method, path, role string) bool
}

// Rules will store the rule configuration
type Rules map[string]map[string][]string

func NewRoleControl() Rules {
	r := Rules{}

	r.AddRule(newRule("/v1/user/", "POST", "admin"))
	r.AddRule(newRule("/v1/user/:id", "GET", "admin"))
	r.AddRule(newRule("/v1/user/drivers", "GET", "admin"))

	r.AddRule(newRule("/v1/travel/", "POST", "admin"))
	r.AddRule(newRule("/v1/travel/:id", "GET", "admin"))
	r.AddRule(newRule("/v1/travel/:id", "GET", "driver"))
	r.AddRule(newRule("/v1/travel/:id", "PUT", "driver"))
	r.AddRule(newRule("/v1/travel/:id", "PUT", "admin"))

	return r
}

func (r Rules) AddRule(rule rule) {
	if _, ok := r[rule.method]; !ok {
		r[rule.method] = map[string][]string{}
	}
	if _, ok := r[rule.method][rule.url]; !ok {
		r[rule.method][rule.url] = []string{}
	}

	r[rule.method][rule.url] = append(r[rule.method][rule.url], rule.role)
}

func (r Rules) CanAccess(method, path, role string) bool {
	if _, exist := r[method]; !exist {
		return false
	}

	if _, exist := r[method][path]; !exist {
		return false
	}

	rolesAccepted := r[method][path]
	for _, roleAccepted := range rolesAccepted {
		if roleAccepted == role {
			return true
		}
	}

	return false
}

// AuthorizeRequest get the user who is authenticated from context, and check if it can
// access to the resource (endpoint and action)
func AuthorizeRequest(rules Ruler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		claimsCtx, exist := ctx.Get("user_on_call")
		if !exist {
			log.Error(ctx, "there was an error getting logged in user from context on authorize request")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
				Code:        "authorize_failure",
				Description: "cannot authorize user",
			})
			return
		}

		claims := claimsCtx.(jwt.Claims)

		if !rules.CanAccess(ctx.Request.Method, ctx.FullPath(), claims.Role) {
			log.Info(ctx, "the user who was logged in cannot access resource",
				log.Int64("user_id", claims.UserID),
				log.String("resource", ctx.FullPath()),
				log.String("role", claims.Role))
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, apiError{
				Code: "authorize_failure",
				Description: fmt.Sprintf("cannot authorize user with role: %s on %s to %s",
					claims.Role, ctx.Request.Method, ctx.Request.URL.Path),
			})
			return
		}
	}
}
