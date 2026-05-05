package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-microservice-app/internal/auth"
	"go-microservice-app/internal/cache"
	"go-microservice-app/internal/config"
	"go-microservice-app/internal/handlers"
	"go-microservice-app/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// Инициализация PostgreSQL
	if err := storage.InitPG(cfg.PGConnStr); err != nil {
		log.Fatalf("Failed to init PostgreSQL: %v", err)
	}
	defer storage.PGPool.Close()

	// Инициализация MongoDB
	mongoClient, err := storage.InitMongo(cfg.MongoURI)
	if err != nil {
		log.Fatalf("Failed to init MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	// Инициализация Redis
	if err := cache.InitRedis(cfg.RedisAddr); err != nil {
		log.Fatalf("Failed to init Redis: %v", err)
	}
	defer cache.Rdb.Close()

	// Инициализация JWT сервиса
	auth.InitJWT(cfg.JWTSecret)

	// Настройка Gin
	router := gin.Default()

	// Открытый маршрут для проверки балансировки
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"server": cfg.ServerID})
	})

	// Группа API
	api := router.Group("/api")
	{
		// Аутентификация
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", handlers.Register)
			authGroup.POST("/login", handlers.Login)
			authGroup.POST("/refresh", handlers.RefreshToken)
			authGroup.GET("/me", auth.AuthRequired(), auth.RoleRequired("user", "seller", "admin"), handlers.Me)
		}

		// Пользователи (только admin)
		usersGroup := api.Group("/users")
		usersGroup.Use(auth.AuthRequired(), auth.RoleRequired("admin"))
		{
			usersGroup.GET("", handlers.GetUsers)
			usersGroup.GET("/:id", handlers.GetUser)
			usersGroup.POST("", handlers.CreateUser)
			usersGroup.PATCH("/:id", handlers.UpdateUser)
			usersGroup.DELETE("/:id", handlers.DeleteUser)
		}

		// Продукты (доступны всем авторизованным, но изменение - admin/seller)
		productsGroup := api.Group("/products")
		productsGroup.Use(auth.AuthRequired())
		{
			productsGroup.GET("", handlers.GetProducts)
			productsGroup.GET("/:id", handlers.GetProduct)
			productsGroup.POST("", auth.RoleRequired("admin", "seller"), handlers.CreateProduct)
			productsGroup.PATCH("/:id", auth.RoleRequired("admin", "seller"), handlers.UpdateProduct)
			productsGroup.DELETE("/:id", auth.RoleRequired("admin"), handlers.DeleteProduct)
		}
	}

	// Запуск сервера
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	log.Println("Server exiting")
}
