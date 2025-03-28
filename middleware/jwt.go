package middleware

import (
	"errors"

	"yt-converter-api/config"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func JWTProtected() fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(config.LoadConfig().JwtSecret)},
		ContextKey: "jwt",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Comprobar si el error es por token expirado
			if errors.Is(err, jwt.ErrTokenExpired) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Tu sesión ha expirado, por favor inicia sesión nuevamente",
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No estás autorizado para acceder a este recurso",
			})
		},
	})
}
