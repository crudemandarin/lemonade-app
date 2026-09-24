package controller

import (
	"errors"
	"net/http"
	"strconv"

	"lemonade-api/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SampleController struct {
	Service *service.SampleService
}

func NewSampleController(service *service.SampleService) *SampleController {
	return &SampleController{Service: service}
}

type sampleRequest struct {
	Name string `json:"name"`
}

func (ctrl *SampleController) List(c *gin.Context) {
	samples, err := ctrl.Service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, samples)
}

func (ctrl *SampleController) Get(c *gin.Context) {
	id, err := parseSampleID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sample, err := ctrl.Service.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "sample not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sample)
}

func (ctrl *SampleController) Create(c *gin.Context) {
	var req sampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	sample, err := ctrl.Service.Create(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, sample)
}

func (ctrl *SampleController) Update(c *gin.Context) {
	id, err := parseSampleID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req sampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	sample, err := ctrl.Service.Update(id, req.Name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "sample not found"})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sample)
}

func (ctrl *SampleController) Delete(c *gin.Context) {
	id, err := parseSampleID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.Service.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func parseSampleID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, errors.New("invalid id")
	}

	return uint(id), nil
}
