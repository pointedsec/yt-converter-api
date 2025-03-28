package pkg

import (
	"errors"
	"fmt"
	"time"

	"yt-converter-api/config"

	"github.com/golang-jwt/jwt/v5"
)

// Tiempo de expiración del token (ejemplo: 24 horas)
const TokenExpiration = time.Hour * 24

func GenerateToken(id string, role string) (string, error) {
	// Convertir la clave JWT a bytes
	secret := []byte(config.LoadConfig().JwtSecret)

	// Tiempo actual y expiración
	now := time.Now()
	expirationTime := now.Add(TokenExpiration)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": id,
		"role":    role,
		"iat":     now.Unix(),            // Emitido en (Issued At)
		"exp":     expirationTime.Unix(), // Expira en
	})

	t, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return t, nil
}

func VerifyToken(tokenString string) (bool, error) {
	secret := []byte(config.LoadConfig().JwtSecret)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return false, fmt.Errorf("token expirado")
		}
		return false, err
	}

	return token.Valid, nil
}

// GetUserFromToken extrae la información del usuario del token JWT
func GetUserFromToken(tokenString string) (string, string, error) {
	secret := []byte(config.LoadConfig().JwtSecret)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", "", fmt.Errorf("token expirado")
		}
		return "", "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, _ := claims["user_id"].(string)
		role, _ := claims["role"].(string)
		return userID, role, nil
	}

	return "", "", jwt.ErrSignatureInvalid
}
