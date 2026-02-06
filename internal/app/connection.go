package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

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
