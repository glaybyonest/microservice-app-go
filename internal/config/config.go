package config

import "os"

type Config struct {
	ServerID  string
	PGConnStr string
	MongoURI  string
	RedisAddr string
	JWTSecret string
	Port      string
}

func Load() *Config {
	return &Config{
		ServerID:  getEnv("SERVER_ID", "backend-1"),
		PGConnStr: getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/usersdb?sslmode=disable"),
		MongoURI:  getEnv("MONGO_URI", "mongodb://localhost:27017"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		JWTSecret: getEnv("JWT_SECRET", "supersecret"),
		Port:      getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
