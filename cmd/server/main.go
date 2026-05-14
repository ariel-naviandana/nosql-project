package main

import (
	"context"
	"log"
	"time"

	"banking-nosql/internal/config"
	"banking-nosql/internal/handlers"
	"banking-nosql/internal/middleware"
	"banking-nosql/internal/repository"
	"banking-nosql/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg := config.Load()

	// ===== CONNECT MONGODB =====
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("Gagal koneksi MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB tidak bisa di-ping: %v", err)
	}
	log.Println("✅ MongoDB terhubung")

	mongoDB := mongoClient.Database(cfg.MongoDB)

	// ===== CONNECT REDIS =====
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       0,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Gagal koneksi Redis: %v", err)
	}
	log.Println("✅ Redis terhubung")

	// ===== SETUP REPOSITORIES =====
	mongoRepo := repository.NewMongoRepository(mongoDB)
	redisRepo := repository.NewRedisRepository(redisClient)

	// Setup indexes MongoDB
	if err := mongoRepo.SetupIndexes(context.Background()); err != nil {
		log.Printf("Warning: gagal setup indexes: %v", err)
	}
	log.Println("✅ MongoDB indexes siap")

	// ===== SETUP SERVICES =====
	authSvc := services.NewAuthService(mongoRepo, redisRepo, cfg.JWTSecret)

	// ===== SETUP ROUTER =====
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"mongodb": "connected",
			"redis":   "connected",
		})
	})

	h := handlers.New(mongoRepo, redisRepo, authSvc)
	authMiddleware := middleware.AuthMiddleware(authSvc, mongoRepo)

	// ===== ROUTES =====

	api := r.Group("/api/v1")
	{
		// Auth (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
			auth.POST("/logout", authMiddleware, h.Logout)
		}

		// Nasabah (authenticated)
		nasabah := api.Group("/nasabah", authMiddleware)
		{
			nasabah.GET("/profile", h.GetProfile)
			nasabah.PUT("/profile", h.UpdateProfile)
			nasabah.GET("/kyc", h.GetMyKYC)
			nasabah.POST("/kyc", h.UploadKYC)
			nasabah.GET("/audit-log", h.GetAuditLog)
			nasabah.GET("/session-info", h.GetSessionInfo)
		}

		// Admin routes (authenticated)
		admin := api.Group("/admin", authMiddleware)
		{
			admin.GET("/nasabah", h.GetAllNasabah)
			admin.PUT("/kyc/:id/verify", h.VerifyKYC)
			admin.GET("/audit-log", h.GetAllAuditLog)
		}
	}

	log.Printf("🚀 Server berjalan di port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}
