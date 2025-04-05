package controllers

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type controller struct {
	router *gin.RouterGroup
	db     *gorm.DB
}

func NewController(router *gin.RouterGroup, db *gorm.DB) *controller {
	if router == nil {
		log.Fatalln("route group should not be nil")
		return nil
	}
	return &controller{router: router, db: db}
}
