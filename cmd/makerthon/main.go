package main

import (
	"log"
	"makerthon/controllers"
	"makerthon/controllers/middleware"
	"makerthon/models"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	db, err := gorm.Open(postgres.Open(os.Getenv("POSTGRES_DNS")), &gorm.Config{
		Logger: logger.Default,
	})
	if err != nil {
		log.Fatalln("an error happened when connecting to the database", err)
		return
	}

	if err := db.AutoMigrate(&models.User{}, &models.Quiz{}, &models.Question{}, &models.Option{}); err != nil {
		log.Fatalln("Error migrating database:", err)
		return
	} else {
		log.Println("Database migration successful")
	}

	r := gin.Default()
	r.Static("/public", "public")

	r.Use(middleware.OptionalAuth(db))

	r.GET("", func(ctx *gin.Context) { ctx.Redirect(http.StatusPermanentRedirect, "/home") })

	ctls := map[string]controllers.Controller{
		"/home": controllers.NewHomeController(),
		"/auth": controllers.NewAuthController(db),
		"/quiz": controllers.NewQuizController(db),
	}

	for route, ctl := range ctls {
		ctl.LoadRoutes(r.Group(route))
	}

	r.GET("/create-quiz", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusFound, "/quiz/create")
	})

	r.Run(os.Getenv("SERVER_PORT"))
}

