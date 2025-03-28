package routes

import (
	"net/http"
	"yt-converter-api/db"
	"yt-converter-api/models"
	"yt-converter-api/pkg"

	"github.com/gofiber/fiber/v2"
)

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"` // admin o guest
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

// Register registra un nuevo usuario en base a usuario y contraseña
func Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"message": err.Error(),
		})
	}
	user := models.User{
		Username: req.Username,
		Password: pkg.GeneratePassword(req.Password),
		Role:     req.Role,
	}
	err := db.DB.QueryRow("INSERT INTO users (username, password, role) VALUES (?, ?, ?)", user.Username, user.Password, user.Role)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al registrar el usuario",
		})
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "Usuario registrado correctamente, puede iniciar sesión",
	})
}

// Login devuelve un token de autenticación
func Login(c *fiber.Ctx) error {
	var request loginRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Error al analizar el cuerpo de la solicitud",
		})
	}
	var user models.User
	// Recuperar el usuario de la base de datos
	err := db.DB.QueryRow("SELECT id, username, password, role FROM users WHERE username = ?", request.Username).Scan(&user.ID, &user.Username, &user.Password, &user.Role)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Usuario o contraseña incorrectos",
		})
	}
	// Verificar la contraseña
	if !pkg.ComparePassword(user.Password, request.Password) {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Usuario o contraseña incorrectos",
		})
	}
	// Generar un token JWT
	token, err := pkg.GenerateToken(user.ID, user.Role)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error":      "Error al generar el token",
			"errorTrace": err.Error(),
		})
	}
	return c.JSON(authResponse{Token: token})
}

func Logout(c *fiber.Ctx) error {
	return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
		"error": "Error al generar el token",
	})
}
