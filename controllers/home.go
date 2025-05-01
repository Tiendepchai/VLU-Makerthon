package controllers

import (
	"encoding/json"
	"makerthon/models"
	"makerthon/views"
	"net/http"

	"github.com/gin-gonic/gin"
)

type homeController struct{}

func NewHomeController() *homeController { return &homeController{} }

func (hc *homeController) LoadRoutes(router *gin.RouterGroup) {
	assertRouteGroup(router, "home")

	router.GET("", hc.GetHomePage)
	router.GET("/dog-image", hc.GetDogImage)
}

func (hc *homeController) GetHomePage(ctx *gin.Context) {
	if url, err := getDogImageUrl(); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	} else {
		ctx.Set("src", url)
		views.Home().Render(ctx, ctx.Writer)
	}
}

func (hc *homeController) GetDogImage(ctx *gin.Context) {
	if url, err := getDogImageUrl(); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	} else {
		ctx.Set("src", url)
	}
}

func getDogImageUrl() (string, error) {
	resp, err := http.Get("https://dog.ceo/api/breeds/image/random")

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	dog := models.DogImage{}

	if err = json.NewDecoder(resp.Body).Decode(&dog); err != nil {
		return "", err
	}

	return dog.Message, nil
}
