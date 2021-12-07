package main

import (
	"github.com/gin-gonic/gin"
	"github.com/nicocarolo/space-drivers/cmd/api/handlers"
	"github.com/nicocarolo/space-drivers/internal/user"
)

func main() {
	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	st, err := user.NewRepository()
	if err != nil {
		panic(err)
	}

	userHandler := handlers.UserHandler{
		Users: user.NewUserStorage(st),
	}

	router.GET("/user/:id", userHandler.Get)
	router.POST("/user/", userHandler.Create)

	router.Run(":8088")
}
