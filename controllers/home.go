package controllers

import (
	"makerthon/views"

	"github.com/gin-gonic/gin"
)

type homeController struct{}

func NewHomeController() *homeController { return &homeController{} }

func (hc *homeController) LoadRoutes(router *gin.RouterGroup) {
	assertRouteGroup(router, "home")

	router.GET("", hc.GetHomePage)
}

func (hc *homeController) GetHomePage(ctx *gin.Context) {
	views.Home().Render(ctx, ctx.Writer)
}
