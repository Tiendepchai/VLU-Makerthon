package main

import (
	"log"
	"makerthon/controllers"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	_, err := gorm.Open(postgres.Open(os.Getenv("POSTGRES_DNS")), &gorm.Config{
		Logger: logger.Default,
	})
	if err != nil {
		log.Fatalln("an error happened when connecting to the database", err)
		return
	}

	r := gin.Default()
	r.Static("/public", "./public")

	r.GET("", func(ctx *gin.Context) { ctx.Redirect(http.StatusPermanentRedirect, "/home") })
	controllers.NewHomeController(controllers.NewController(r.Group("/home"), nil)).LoadRoutes()

	r.Run(os.Getenv("SERVER_PORT"))
}
