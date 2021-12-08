package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nicocarolo/space-drivers/cmd/api/handlers"
	"github.com/nicocarolo/space-drivers/internal/platform/metrics"
	"github.com/nicocarolo/space-drivers/internal/travel"
	"github.com/nicocarolo/space-drivers/internal/user"
	"net/http"
	"time"
)

// Config for api
type Config struct {
	userHandler   handlers.UserHandler
	travelHandler handlers.TravelHandler
	authHandler   handlers.AuthHandler

	ruler handlers.Ruler
}

func main() {
	setApi(getConfig())
}

// getConfig return api configuration with handlers
func getConfig() Config {
	userStorage, err := user.NewRepository()
	if err != nil {
		panic(err)
	}

	travelStorage, err := travel.NewRepository()
	if err != nil {
		panic(err)
	}

	userHandler := handlers.UserHandler{
		Users: user.NewUserStorage(userStorage),
	}

	travelHandler := handlers.TravelHandler{
		Users:   user.NewUserStorage(userStorage),
		Travels: travel.NewTravelStorage(travelStorage),
	}

	authHandler := handlers.AuthHandler{
		Users: user.NewUserStorage(userStorage),
	}

	rules := handlers.NewRoleControl()

	return Config{
		userHandler:   userHandler,
		travelHandler: travelHandler,
		authHandler:   authHandler,
		ruler:         rules,
	}
}

// setApi configure api on gin router and run
func setApi(config Config) {
	router := gin.Default()

	router.Use(gin.CustomRecovery(panicRecover))
	router.Use(trace())

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	v1 := router.Group("/v1")

	v1.GET("/users/:id", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(config.ruler), config.userHandler.Get)
	v1.POST("/users", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(config.ruler), config.userHandler.Create)
	v1.GET("/users/drivers", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(config.ruler), config.userHandler.GetDrivers)

	v1.GET("/travels/:id", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(config.ruler), config.travelHandler.Get)
	v1.PUT("/travels/:id", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(config.ruler), config.travelHandler.Edit)
	v1.POST("/travels", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(config.ruler), config.travelHandler.Create)

	v1.POST("/login", config.authHandler.Login)

	err := router.Run(":8080")
	if err != nil {
		panic("cannot run router")
	}
}

// panicRecover
func panicRecover(c *gin.Context, err interface{}) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"code":   "unexpected_error",
		"detail": err,
	})
}

// trace metric for endpoint time elapsed and http status code count
func trace() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		const (
			timeEndpointMetric  = "application.space.api.time"
			countEndpointMetric = "application.space.api.count"
		)
		start := time.Now()

		ctx.Next()

		// track time
		elapsed := time.Since(start)
		metrics.Timing(ctx, timeEndpointMetric, elapsed, []string{
			"endpoint", ctx.FullPath(),
			"http_status_code", fmt.Sprintf("%d", ctx.Writer.Status()),
		})

		// track occurrences
		metrics.Inc(ctx, countEndpointMetric, []string{
			"endpoint", ctx.FullPath(),
			"http_status_code", fmt.Sprintf("%d", ctx.Writer.Status()),
		})
	}
}
