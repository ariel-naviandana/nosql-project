package config

import "os"

type Config struct {
	MongoURI    string
	MongoDB     string
	RedisAddr   string
	RedisPass   string
	JWTSecret   string
	Port        string
}

func Load() *Config {
	return &Config{
		MongoURI:  getEnv("MONGODB_URI", "mongodb://admin:password123@localhost:27017/banking_db?authSource=admin"),
		MongoDB:   getEnv("MONGODB_DATABASE", "banking_db"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass: getEnv("REDIS_PASSWORD", "password123"),
		JWTSecret: getEnv("JWT_SECRET", "super-secret-jwt-key-banking-2024"),
		Port:      getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
