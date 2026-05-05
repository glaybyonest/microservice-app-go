package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"go-microservice-app/internal/auth"
	"go-microservice-app/internal/storage"
)

type RegisterInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.Role == "" {
		input.Role = "user"
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	_, err = storage.PGPool.Exec(c.Request.Context(),
		"INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)",
		input.Username, string(hashed), input.Role)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "User registered"})
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var id int
	var username, passwordHash, role string
	var blocked bool
	err := storage.PGPool.QueryRow(c.Request.Context(),
		"SELECT id, username, password_hash, role, blocked FROM users WHERE username=$1",
		input.Username).Scan(&id, &username, &passwordHash, &role, &blocked)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if blocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "User is blocked"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	accessToken, _ := auth.GenerateAccessToken(username, role)
	refreshToken, _ := auth.GenerateRefreshToken(username, role)

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	})
}

func RefreshToken(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	claims, err := auth.ValidateRefreshToken(body.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Проверим, существует ли пользователь и не заблокирован
	var blocked bool
	err = storage.PGPool.QueryRow(c.Request.Context(),
		"SELECT blocked FROM users WHERE username=$1", claims.Username).Scan(&blocked)
	if err != nil || blocked {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found or blocked"})
		return
	}

	auth.InvalidateRefreshToken(body.RefreshToken)
	newAccess, _ := auth.GenerateAccessToken(claims.Username, claims.Role)
	newRefresh, _ := auth.GenerateRefreshToken(claims.Username, claims.Role)
	c.JSON(http.StatusOK, gin.H{
		"accessToken":  newAccess,
		"refreshToken": newRefresh,
	})
}

func Me(c *gin.Context) {
	claims, _ := c.Get("userClaims")
	userClaims := claims.(*auth.Claims)
	c.JSON(http.StatusOK, gin.H{
		"username": userClaims.Username,
		"role":     userClaims.Role,
	})
}
