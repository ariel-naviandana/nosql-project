package middleware

import (
	"net/http"
	"strings"

	"banking-nosql/internal/models"
	"banking-nosql/internal/repository"
	"banking-nosql/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func AuthMiddleware(authSvc *services.AuthService, mongoRepo *repository.MongoRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token tidak ditemukan"})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "format token tidak valid"})
			c.Abort()
			return
		}

		nasabahID, err := authSvc.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		objID, err := primitive.ObjectIDFromHex(nasabahID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "session tidak valid"})
			c.Abort()
			return
		}

		nasabah, err := mongoRepo.FindNasabahByID(c.Request.Context(), objID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "nasabah tidak ditemukan"})
			c.Abort()
			return
		}

		c.Set("nasabah", nasabah)
		c.Set("token", token)
		c.Next()
	}
}

func RateLimitMiddleware(redisRepo *repository.RedisRepository, maxReq int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		count, err := redisRepo.IncrRateLimit(c.Request.Context(), "api:"+ip, 60*1000000000)
		if err != nil || count > maxReq {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "terlalu banyak request"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func GetNasabah(c *gin.Context) *models.Nasabah {
	nasabah, _ := c.Get("nasabah")
	return nasabah.(*models.Nasabah)
}

func GetToken(c *gin.Context) string {
	token, _ := c.Get("token")
	return token.(string)
}
