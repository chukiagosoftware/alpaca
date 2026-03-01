package aisearch

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests
type Handler struct {
	service *AISearchService
}

// NewHandler creates a new handler
func NewHandler(service *AISearchService) *Handler {
	return &Handler{service: service}
}

// SetupRoutes sets up the routes
func (h *Handler) SetupRoutes(r *gin.Engine) {
	r.POST("/upload/cities", h.uploadCities)
	r.POST("/upload/hotels", h.uploadHotels)
	r.POST("/upload/reviews", h.uploadReviews)
	r.POST("/transform", h.transform)
	r.POST("/vectorize", h.vectorize)
	r.POST("/search", h.search)
	r.POST("/llm", h.llm)
}

// uploadCities
func (h *Handler) uploadCities(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cities, ok := req.Data.([]City)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data format"})
		return
	}
	if err := h.service.UploadCities(c.Request.Context(), cities); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "uploaded"})
}

// Similar for hotels and reviews
func (h *Handler) uploadHotels(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hotels, ok := req.Data.([]Hotel)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data format"})
		return
	}
	if err := h.service.UploadHotels(c.Request.Context(), hotels); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "uploaded"})
}

func (h *Handler) uploadReviews(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reviews, ok := req.Data.([]Review)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data format"})
		return
	}
	if err := h.service.UploadReviews(c.Request.Context(), reviews); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "uploaded"})
}

// transform
func (h *Handler) transform(c *gin.Context) {
	if err := h.service.TransformData(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "transformed"})
}

// vectorize
func (h *Handler) vectorize(c *gin.Context) {
	if err := h.service.VectorizeData(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "vectorized"})
}

// search
func (h *Handler) search(c *gin.Context) {
	var req VectorSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	results, err := h.service.VectorSearch(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, VectorSearchResponse{Results: results})
}

// llm
func (h *Handler) llm(c *gin.Context) {
	var req LLMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	answer, err := h.service.GenerateLLMResponse(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, LLMResponse{Answer: answer})
}
