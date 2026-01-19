package cloudinary

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

//go:generate mockgen -source=cloudinary_service.go -destination=../mock/cloudinary/cloudinary_service_mock.go -package=mock
type Service interface {
	UploadImage(ctx context.Context, file multipart.File, filename string) (string, error)
	DeleteImage(ctx context.Context, publicID string) error
}

type service struct {
	cld    *cloudinary.Cloudinary
	folder string
}

func NewService(cloudName, apiKey, apiSecret, folder string) (Service, error) {
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudinary: %w", err)
	}

	return &service{
		cld:    cld,
		folder: folder,
	}, nil
}

// UploadImage uploads an image to Cloudinary and returns the secure URL
func (s *service) UploadImage(ctx context.Context, file multipart.File, filename string) (string, error) {
	uploadResult, err := s.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:         s.folder,
		PublicID:       filename,
		ResourceType:   "image",
		Transformation: "c_fill,w_800,h_800,q_auto",
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	return uploadResult.SecureURL, nil
}

// DeleteImage deletes an image from Cloudinary
func (s *service) DeleteImage(ctx context.Context, publicID string) error {
	_, err := s.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}

// Helper function to extract public ID from Cloudinary URL
func ExtractPublicID(url, folder string) string {
	// Example URL: https://res.cloudinary.com/demo/image/upload/v1234567890/folder/filename.jpg
	// Extract: folder/filename
	// This is a simplified version, adjust based on your URL structure
	// You might need to use regex or string manipulation based on your needs
	return "" // Implement based on your URL structure
}
