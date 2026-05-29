package upload

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopickup/internal/config"
	"gopickup/internal/storage"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp" // register webp decoder for image.Decode
)

const (
	maxUploadBytes = 10 << 20 // 10 MB
	maxImageWidth  = 1200     // downscale anything wider than this
	jpegQuality    = 82
)

type UploadHandler struct {
	cfg *config.Config
	r2  *storage.R2Client // nil when R2 is not configured (local-disk fallback)
}

func NewUploadHandler(cfg *config.Config) *UploadHandler {
	h := &UploadHandler{cfg: cfg}
	if cfg.R2Enabled() {
		if client, err := storage.NewR2Client(cfg); err == nil {
			h.r2 = client
		}
	}
	return h
}

func (h *UploadHandler) UploadFile(c *gin.Context) {
	// Require authentication
	if _, exists := c.Get("userID"); !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context"})
		return
	}

	// Accept "image" (primary) or "file" (the Flutter profile upload uses this).
	fileHeader, err := c.FormFile("image")
	if err != nil {
		fileHeader, err = c.FormFile("file")
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File parsing failed: " + err.Error()})
		return
	}

	if fileHeader.Size > maxUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image too large (max 10MB)"})
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only image files (.jpg, .jpeg, .png, .webp) are allowed"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file: " + err.Error()})
		return
	}
	defer src.Close()

	raw, err := io.ReadAll(io.LimitReader(src, maxUploadBytes+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Resize + re-encode to a compressed JPEG. imaging.Decode also applies EXIF
	// orientation. If decoding fails (corrupt/unsupported), fall back to the
	// original bytes so the upload still succeeds.
	data := raw
	contentType := "image/jpeg"
	outExt := ".jpg"
	if img, derr := imaging.Decode(bytes.NewReader(raw), imaging.AutoOrientation(true)); derr == nil {
		if img.Bounds().Dx() > maxImageWidth {
			img = imaging.Resize(img, maxImageWidth, 0, imaging.Lanczos)
		}
		var buf bytes.Buffer
		if jerr := imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(jpegQuality)); jerr == nil {
			data = buf.Bytes()
		}
	} else {
		contentType = mimeForExt(ext)
		outExt = ext
	}

	key := uuid.New().String() + outExt

	// Prefer R2 (durable, survives redeploys). Fall back to local disk when R2
	// is not configured (e.g. local development).
	if h.r2 != nil {
		url, uerr := h.r2.Upload(context.Background(), key, data, contentType)
		if uerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload failed: " + uerr.Error()})
			return
		}
		respondOK(c, url)
		return
	}

	// Local-disk fallback (matches the previous behaviour).
	savePath := filepath.Join("uploads", key)
	if werr := os.WriteFile(savePath, data, 0o644); werr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + werr.Error()})
		return
	}

	baseURL := strings.TrimSuffix(h.cfg.AppURL, "/")
	if strings.Contains(baseURL, "localhost") && !strings.Contains(c.Request.Host, "localhost") {
		scheme := "http"
		if c.Request.Header.Get("X-Forwarded-Proto") == "https" || c.Request.TLS != nil {
			scheme = "https"
		}
		baseURL = fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	}
	respondOK(c, fmt.Sprintf("%s/uploads/%s", baseURL, key))
}

// respondOK returns the uploaded URL under both keys so any client
// (some read "image_url", the Flutter profile screen reads "url") works.
func respondOK(c *gin.Context, url string) {
	c.JSON(http.StatusOK, gin.H{
		"message":   "File uploaded successfully",
		"image_url": url,
		"url":       url,
	})
}

func mimeForExt(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
