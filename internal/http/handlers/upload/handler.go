package upload

import (
	"fmt"
	"gopickup/internal/config"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UploadHandler struct {
	cfg *config.Config
}

func NewUploadHandler(cfg *config.Config) *UploadHandler {
	return &UploadHandler{cfg: cfg}
}

func (h *UploadHandler) UploadFile(c *gin.Context) {
	// Require authentication
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Retrieve file from form data
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File parsing failed: " + err.Error()})
		return
	}

	// Validate extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only image files (.jpg, .jpeg, .png, .webp) are allowed"})
		return
	}

	// Generate unique filename
	fileName := uuid.New().String() + ext
	savePath := filepath.Join("uploads", fileName)

	// Save file locally
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
		return
	}

	// Construct full file URL
	// If AppURL is "https://api.example.com", fileURL becomes "https://api.example.com/uploads/uuid.png"
	baseURL := strings.TrimSuffix(h.cfg.AppURL, "/")
	// Since static route is r.Static("/uploads", ...), the path is /uploads/...
	fileURL := fmt.Sprintf("%s/uploads/%s", baseURL, fileName)

	c.JSON(http.StatusOK, gin.H{
		"message":   "File uploaded successfully",
		"image_url": fileURL,
	})
}
