package main

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

var jwtSecret = []byte(getSecret())

func getSecret() string {
	sec := os.Getenv("JWT_SECRET")
	if sec == "" {
		// Default secret for local standalone store operation
		sec = "fainaya-store-support-local-secret-key-32b!"
	}
	return sec
}

type JWTCustomClaims struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	FullName  string `json:"full_name"`
	Role      string `json:"role"`
	StoreCode string `json:"store_code"`
	jwt.RegisteredClaims
}

func generateToken(user User) (string, error) {
	claims := &JWTCustomClaims{
		UserID:    user.ID,
		Username:  user.Username,
		FullName:  user.FullName,
		Role:      user.Role,
		StoreCode: user.StoreCode,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func authMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		var tokenString string

		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			cookie, err := c.Cookie("fainaya_token")
			if err == nil && cookie.Value != "" {
				tokenString = cookie.Value
			}
		}

		if tokenString == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Akses ditolak: Token autentikasi tidak ditemukan"})
		}

		claims := &JWTCustomClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("Metode tanda tangan token tidak valid")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Sesi tidak valid atau telah kedaluwarsa"})
		}

		c.Set("user", claims)
		return next(c)
	}
}

// requireRoles checks if user has one of allowed roles
func requireRoles(allowedRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := c.Get("user").(*JWTCustomClaims)
			if !ok || claims == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Pengguna belum terautentikasi"})
			}

			// super_admin always has access
			if claims.Role == "super_admin" {
				return next(c)
			}

			for _, role := range allowedRoles {
				if claims.Role == role {
					return next(c)
				}
			}

			return c.JSON(http.StatusForbidden, map[string]string{"error": "Akses ditolak: Peran Anda tidak memiliki izin untuk fitur ini"})
		}
	}
}
