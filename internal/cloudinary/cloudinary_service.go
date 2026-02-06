package cloudinary

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

//go:generate mockgen -source=cloudinary_service.go -destination=../mock/cloudinary/cloudinary_service_mock.go -package=mock
type Service interface {
	UploadImage(ctx context.Context, file multipart.File, filename string, folderName string) (string, error)
	DeleteImage(ctx context.Context, publicID string) error
}

type service struct {
	cld *cloudinary.Cloudinary
}

func NewService(cloudName, apiKey, apiSecret string) (Service, error) {
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudinary: %w", err)
	}

	return &service{
		cld: cld,
	}, nil
}

// UploadImage uploads an image to Cloudinary and returns the secure URL
func (s *service) UploadImage(ctx context.Context, file multipart.File, filename string, folderName string) (string, error) {
	log.Println("Upload IMAGE")
	log.Println(folderName)
	uploadResult, err := s.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:         folderName,
		PublicID:       filename,
		ResourceType:   "image",
		Transformation: "c_fill,w_800,h_800,q_auto",
	})
	log.Println(uploadResult)
	log.Println(err)
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	return uploadResult.SecureURL, nil
}

// DeleteImage deletes an image from Cloudinary
func (s *service) DeleteImage(ctx context.Context, publicID string) error {
	log.Println("[cloudinary][delete] publicID:", publicID)

	res, err := s.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})
	log.Println("[cloudinary][delete] result:", res)

	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}

func ExtractPublicID(imageURL, folder string) (string, error) {
	log.Println("[cloudinary][extract] imageURL:", imageURL)

	u, err := url.Parse(imageURL)
	if err != nil {
		return "", err
	}

	parts := strings.Split(u.Path, "/image/upload/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid cloudinary url")
	}

	// parts[1] = v1769161193/go-gadget/brands/brand-xxx.jpg
	path := parts[1]

	// remove version
	pathParts := strings.SplitN(path, "/", 2)
	if len(pathParts) != 2 {
		return "", fmt.Errorf("invalid cloudinary path")
	}

	// remove extension
	publicID := strings.TrimSuffix(pathParts[1], filepath.Ext(pathParts[1]))

	log.Println("[cloudinary][extract] final publicID:", publicID)
	return publicID, nil
}
