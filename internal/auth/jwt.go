package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret  []byte
	refreshMap = make(map[string]string) // упрощённое хранение refresh-токенов (в реальном проекте – Redis)
)

const (
	AccessTokenExpiry  = 15 * time.Minute
	RefreshTokenExpiry = 7 * 24 * time.Hour
)

func InitJWT(secret string) {
	jwtSecret = []byte(secret)
}

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(username, role string) (string, error) {
	claims := Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   username,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func GenerateRefreshToken(username, role string) (string, error) {
	claims := Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(RefreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   username,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	refreshMap[tokenStr] = username
	return tokenStr, nil
}

func ValidateRefreshToken(tokenStr string) (*Claims, error) {
	if _, exists := refreshMap[tokenStr]; !exists {
		return nil, fmt.Errorf("token not found in store")
	}
	claims, err := parseToken(tokenStr)
	if err != nil {
		delete(refreshMap, tokenStr)
		return nil, err
	}
	return claims, nil
}

func InvalidateRefreshToken(tokenStr string) {
	delete(refreshMap, tokenStr)
}

func parseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

// ParseAccessToken парсит и валидирует access-токен
func ParseAccessToken(tokenStr string) (*Claims, error) {
	return parseToken(tokenStr)
}
