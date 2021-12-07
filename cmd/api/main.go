package main

import (
	"github.com/gin-gonic/gin"
	"github.com/nicocarolo/space-drivers/cmd/api/handlers"
	"github.com/nicocarolo/space-drivers/internal/travel"
	"github.com/nicocarolo/space-drivers/internal/user"
	"net/http"
)

func main() {
	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	ust, err := user.NewRepository()
	if err != nil {
		panic(err)
	}

	travelst, err := travel.NewRepository()
	if err != nil {
		panic(err)
	}

	userHandler := handlers.UserHandler{
		Users: user.NewUserStorage(ust),
	}

	travelHandler := handlers.TravelHandler{
		Users:   user.NewUserStorage(ust),
		Travels: travel.NewTravelStorage(travelst),
	}

	authHandler := handlers.AuthHandler{
		Users: user.NewUserStorage(ust),
	}

	rules := handlers.NewRoleControl()

	v1 := router.Group("/v1")
	v1.Use(gin.CustomRecovery(panicRecover))

	v1.GET("/user/:id", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(rules), userHandler.Get)
	v1.POST("/user/", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(rules), userHandler.Create)
	v1.GET("/user/drivers", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(rules), userHandler.GetDrivers)

	v1.GET("/travel/:id", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(rules), travelHandler.Get)
	v1.PUT("/travel/:id", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(rules), travelHandler.Edit)
	v1.POST("/travel/", handlers.AuthenticateRequest(), handlers.AuthorizeRequest(rules), travelHandler.Create)

	v1.POST("/login/", authHandler.Login)

	router.Run(":8088")
}

func panicRecover(c *gin.Context, err interface{}) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"code":   "unexpected_error",
		"detail": err,
	})
}
