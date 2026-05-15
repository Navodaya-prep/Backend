package handlers

import (
	"context"
	"net/http"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/navodayaprime/api/utils"
)

const maxUploadSize = 5 << 20 // 5 MB

var allowedMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// UploadImage handles image upload for questions.
// POST /admin/upload/image — multipart form with field "image"
// Returns the full Cloudinary HTTPS URL.
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

	// Detect content type from first 512 bytes
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])

	if !allowedMimeTypes[contentType] {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_TYPE", "Only JPEG, PNG, GIF, and WebP images are allowed")
		return
	}

	// Reset reader after sniffing
	if _, err := file.Seek(0, 0); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "UPLOAD_ERROR", "Failed to process file")
		return
	}

	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "UPLOAD_ERROR", "Failed to initialise storage")
		return
	}

	result, err := cld.Upload.Upload(context.Background(), file, uploader.UploadParams{
		Folder: "navodaya",
	})
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "UPLOAD_ERROR", "Failed to upload image")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"url": result.SecureURL}, "Image uploaded")
}
