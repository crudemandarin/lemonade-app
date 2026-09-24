package main

import (
	"log"
	"net/http"

	"lemonade-api/controller"
	"lemonade-api/internal/api"
	"lemonade-api/libraries"
	"lemonade-api/model"
	"lemonade-api/repository"
	"lemonade-api/service"

	"github.com/gin-gonic/gin"
)

func index(ctx *gin.Context) {
	ctx.String(http.StatusOK, "Hello, world!")
}

func main() {
	secrets := &libraries.Secrets{}
	if err := secrets.Init(); err != nil {
		log.Fatalf("init secrets: %v", err)
	}

	db := &libraries.Database{}
	if err := db.Init(secrets); err != nil {
		log.Fatalf("init database: %v", err)
	}

	if err := db.AutoMigrate(&model.Sample{}); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	sampleRepository := repository.NewSampleRepository(db)
	sampleService := service.NewSampleService(sampleRepository)
	sampleController := controller.NewSampleController(sampleService)

	router := gin.Default()
	router.GET("/", index)
	api.RegisterHealth(router)

	sample := router.Group("/samples")
	sample.GET("", sampleController.List)
	sample.GET("/:id", sampleController.Get)
	sample.POST("", sampleController.Create)
	sample.PUT("/:id", sampleController.Update)
	sample.DELETE("/:id", sampleController.Delete)

	if err := router.Run(); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
