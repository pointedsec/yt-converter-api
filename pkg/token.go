package pkg

import (
	"yt-converter-api/config"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(id string, role string) (string, error) {
	// Convertir la clave JWT a bytes
	secret := []byte(config.LoadConfig().JwtSecret)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": id,
		"role":    role,
	})

	t, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return t, nil
}

func VerifyToken(tokenString string) (bool, error) {
	// Convertir la clave JWT a bytes
	secret := []byte(config.LoadConfig().JwtSecret)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
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
		return "", "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := claims["user_id"].(string)
		role := claims["role"].(string)
		return userID, role, nil
	}

	return "", "", jwt.ErrSignatureInvalid
}
