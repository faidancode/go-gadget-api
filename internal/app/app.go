package app

import (
	"go-gadget-api/internal/cloudinary"
	"os"

	"github.com/gin-gonic/gin"
)

func BuildApp(router *gin.Engine) error {
	// 1. Setup Infrastructure
	db, err := connectDBWithRetry(os.Getenv("DB_URL"), 5)
	if err != nil {
		return err
	}

	redisClient, err := connectRedisWithRetry(os.Getenv("REDIS_ADDR"), 5)
	if err != nil {
		return err
	}
	_ = redisClient

	kafkaWriter, err := connectKafkaWithRetry(os.Getenv("KAFKA_BROKER"), 5)
	if err != nil {
		return err
	}
	_ = kafkaWriter

	// 2. Setup Third Party Services
	cloudinaryService, err := cloudinary.NewService(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		return err
	}

	// 3. Register Modules & Routes
	registerModules(router, db, cloudinaryService)

	return nil
}
