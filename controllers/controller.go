package controllers

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Controller interface {
	LoadRoutes(router *gin.RouterGroup)
}

func assertRouteGroup(router *gin.RouterGroup, name string) {
	if router == nil {
		log.Fatalf("missing router on the %s controller", name)
	}
}

func assertDatabase(db *gorm.DB, name string) {
	if db == nil {
		log.Fatalf("missing db on the %s controller", name)
	}
}
