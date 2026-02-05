package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"gadget-api/internal/auth"
	"gadget-api/internal/brand"
	"gadget-api/internal/cart"
	"gadget-api/internal/category"
	"gadget-api/internal/cloudinary"
	"gadget-api/internal/dbgen"
	"gadget-api/internal/order"
	"gadget-api/internal/product"
	"gadget-api/internal/product/adapters"
	"gadget-api/internal/review"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

func BuildApp(router *gin.Engine) error {
	// --- Infra ---
	db, err := connectDBWithRetry(os.Getenv("DB_URL"), 5)
	if err != nil {
		return err
	}

	redisClient, err := connectRedisWithRetry(os.Getenv("REDIS_ADDR"), 5)
	if err != nil {
		return err
	}

	kafkaWriter, err := connectKafkaWithRetry(os.Getenv("KAFKA_BROKER"), 5)
	if err != nil {
		return err
	}

	_ = redisClient
	_ = kafkaWriter

	queries := dbgen.New(db)

	cloudinaryService, err := cloudinary.NewService(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		return err
	}

	// --- Repositories ---
	authRepo := auth.NewRepository(queries)
	categoryRepo := category.NewRepository(queries)
	brandRepo := brand.NewRepository(queries)
	productRepo := product.NewRepository(queries)
	reviewRepo := review.NewRepository(queries)
	cartRepo := cart.NewRepository(queries)

	// --- Services ---
	authService := auth.NewService(authRepo)

	categoryService := category.NewService(db, categoryRepo, cloudinaryService)
	brandService := brand.NewService(db, brandRepo, cloudinaryService)

	reviewService := review.NewService(
		db,
		reviewRepo,
		productRepo,
	)

	productService := product.NewService(
		db,
		productRepo,
		categoryRepo,
		reviewRepo,
		cloudinaryService,
	)

	cartService := cart.NewService(db, cartRepo)

	// --- Adapters ---
	reviewEligibilityAdapter :=
		adapters.NewReviewEligibilityAdapter(reviewService)

	// --- Handlers ---
	authHandler := auth.NewHandler(authService)
	categoryHandler := category.NewHandler(categoryService)
	brandHandler := brand.NewHandler(brandService)
	reviewHandler := review.NewHandler(reviewService)
	cartHandler := cart.NewHandler(cartService)

	productHandler := product.NewHandler(
		productService,
		reviewEligibilityAdapter,
	)

	// --- Routes ---
	api := router.Group("/api/v1")
	{
		auth.RegisterRoutes(api, authHandler)
		brand.RegisterRoutes(api, brandHandler)
		category.RegisterRoutes(api, categoryHandler)
		product.RegisterRoutes(api, productHandler)
		review.RegisterRoutes(api, reviewHandler)
		cart.RegisterRoutes(api, cartHandler)
		order.RegisterRoutes(api, order.NewHandler(nil))
	}

	return nil
}

func connectDBWithRetry(dsn string, maxRetries int) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 1; i <= maxRetries; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				log.Println("✅ Connected to database")
				return db, nil
			}
		}

		log.Printf("⚠️ DB retry %d/%d failed: %v", i, maxRetries, err)
		time.Sleep(5 * time.Second)
	}

	return nil, err
}

func connectRedisWithRetry(addr string, maxRetries int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	for i := 1; i <= maxRetries; i++ {
		ctx := context.Background()
		if err := rdb.Ping(ctx).Err(); err == nil {
			log.Println("✅ Connected to Redis")
			return rdb, nil
		}

		log.Printf("⚠️ Redis retry %d/%d failed", i, maxRetries)
		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect redis")
}
func connectKafkaWithRetry(broker string, maxRetries int) (*kafka.Writer, error) {
	for i := 1; i <= maxRetries; i++ {
		writer := &kafka.Writer{
			Addr: kafka.TCP(broker),
		}

		conn, err := kafka.Dial("tcp", broker)
		if err == nil {
			conn.Close()
			log.Println("✅ Connected to Kafka")
			return writer, nil
		}

		log.Printf("⚠️ Kafka retry %d/%d failed", i, maxRetries)
		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect kafka")
}
