package controllers

import (
	"encoding/json"
	"log"
	"makerthon/models"
	"makerthon/views"
	"net/http"

	"github.com/gin-gonic/gin"
)

type homeController struct{ *controller }

func NewHomeController(ctl *controller) *homeController {
	return &homeController{controller: ctl}
}

func (hc *homeController) LoadRoutes() {
	router := hc.controller.router
	router.GET("", hc.GetHomePage)
	router.GET("/dog-image", hc.GetDogImage)
}

func (hc *homeController) GetHomePage(ctx *gin.Context) {
	views.Home().Render(ctx, ctx.Writer)
}

func (hc *homeController) GetDogImage(ctx *gin.Context) {
	src, err := getDogImage()
	if err == nil {
		views.DogImage(src, "this is a dog").Render(ctx, ctx.Writer)
		return
	}
	msg := "something went wrong when retrieving dog image"
	ctx.Status(http.StatusInternalServerError)
	ctx.Header("Content-Type", "text/html")
	views.DogImage("", msg).Render(ctx, ctx.Writer)
	log.Println(msg, err)
}

func getDogImage() (string, error) {
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
