package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"go-microservice-app/internal/cache"
	"go-microservice-app/internal/models"
	"go-microservice-app/internal/storage"
)

const usersCacheTTL = 1 * time.Minute

func GetUsers(c *gin.Context) {
	ctx := c.Request.Context()
	cacheKey := "users:all"

	// Попытка из кэша
	var users []models.User
	if found, _ := cache.GetFromCache(cacheKey, &users); found {
		c.JSON(http.StatusOK, gin.H{"source": "cache", "data": users})
		return
	}

	// Из БД
	rows, err := storage.PGPool.Query(ctx, "SELECT id, username, role, blocked, created_at, updated_at FROM users WHERE blocked = false ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Blocked, &u.CreatedAt, &u.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		users = append(users, u)
	}

	if err := cache.SetToCache(cacheKey, users, usersCacheTTL); err != nil {
		fmt.Println("cache set error:", err)
	}
	c.JSON(http.StatusOK, gin.H{"source": "server", "data": users})
}

func GetUser(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	cacheKey := fmt.Sprintf("users:%d", id)
	var user models.User
	if found, _ := cache.GetFromCache(cacheKey, &user); found {
		c.JSON(http.StatusOK, gin.H{"source": "cache", "data": user})
		return
	}

	err = storage.PGPool.QueryRow(ctx,
		"SELECT id, username, role, blocked, created_at, updated_at FROM users WHERE id=$1 AND blocked=false", id).
		Scan(&user.ID, &user.Username, &user.Role, &user.Blocked, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	_ = cache.SetToCache(cacheKey, user, usersCacheTTL)
	c.JSON(http.StatusOK, gin.H{"source": "server", "data": user})
}

type CreateUserInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

func CreateUser(c *gin.Context) {
	ctx := c.Request.Context()
	var input CreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.Role == "" {
		input.Role = "user"
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	_, err := storage.PGPool.Exec(ctx,
		"INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)",
		input.Username, string(hashed), input.Role)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	// Инвалидация кэша списка
	_ = cache.InvalidateCache("users:all")
	c.JSON(http.StatusCreated, gin.H{"message": "User created"})
}

type UpdateUserInput struct {
	Username *string `json:"username"`
	Role     *string `json:"role"`
	Blocked  *bool   `json:"blocked"`
}

func UpdateUser(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	var input UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Динамическое построение запроса
	set := ""
	args := []interface{}{}
	argID := 1

	if input.Username != nil {
		set += fmt.Sprintf("username=$%d,", argID)
		args = append(args, *input.Username)
		argID++
	}
	if input.Role != nil {
		set += fmt.Sprintf("role=$%d,", argID)
		args = append(args, *input.Role)
		argID++
	}
	if input.Blocked != nil {
		set += fmt.Sprintf("blocked=$%d,", argID)
		args = append(args, *input.Blocked)
		argID++
	}
	if set == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}
	set += fmt.Sprintf("updated_at=NOW() WHERE id=$%d", argID)
	args = append(args, id)

	query := "UPDATE users SET " + set
	_, err := storage.PGPool.Exec(ctx, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = cache.InvalidateCache("users:all", fmt.Sprintf("users:%d", id))
	c.JSON(http.StatusOK, gin.H{"message": "User updated"})
}

func DeleteUser(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)

	_, err := storage.PGPool.Exec(ctx, "UPDATE users SET blocked=true WHERE id=$1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = cache.InvalidateCache("users:all", fmt.Sprintf("users:%d", id))
	c.JSON(http.StatusOK, gin.H{"message": "User blocked"})
}
