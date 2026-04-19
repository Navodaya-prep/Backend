package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"navodaya-api/utils"

	"github.com/gin-gonic/gin"
)

const maxUploadSize = 5 << 20 // 5 MB
var allowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// UploadImage handles image upload for questions.
// POST /admin/upload/image — multipart form with field "image"
func UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "NO_FILE", "Image file is required")
		return
	}
	defer file.Close()

	if header.Size > maxUploadSize {
		utils.ErrorRes(c, http.StatusBadRequest, "FILE_TOO_LARGE", "Image must be under 5MB")
		return
	}

	// Detect content type from file header
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])

	ext, ok := allowedMimeTypes[contentType]
	if !ok {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_TYPE", "Only JPEG, PNG, GIF, and WebP images are allowed")
		return
	}

	// Reset reader after sniffing
	if _, err := file.Seek(0, 0); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "UPLOAD_ERROR", "Failed to process file")
		return
	}

	// Generate unique filename
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "UPLOAD_ERROR", "Failed to generate filename")
		return
	}
	filename := hex.EncodeToString(randBytes) + ext

	// Ensure uploads directory exists
	uploadDir := getUploadDir()
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "UPLOAD_ERROR", "Failed to create upload directory")
		return
	}

	destPath := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(header, destPath); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "UPLOAD_ERROR", "Failed to save image")
		return
	}

	// Build public URL
	imageURL := fmt.Sprintf("/uploads/%s", filename)

	utils.Success(c, http.StatusOK, gin.H{"url": imageURL}, "Image uploaded")
}

func getUploadDir() string {
	dir := os.Getenv("UPLOAD_DIR")
	if dir == "" {
		dir = "./uploads"
	}
	return dir
}
