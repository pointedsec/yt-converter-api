package routes

import (
	"log"
	"net/http"
	"strconv"

	"yt-converter-api/config"
	"yt-converter-api/db"
	"yt-converter-api/models"
	"yt-converter-api/pkg"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type UserResponse struct {
	User   models.User
	Videos []models.Video
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

type UpdateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

// GetUsers obtiene todos los usuarios
func GetUsers(c *fiber.Ctx) error {
	rows, err := db.DB.Query("SELECT * FROM users")
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los usuarios",
		})
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.Active)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al obtener los usuarios",
			})
		}
		users = append(users, user)
	}

	return c.JSON(users)
}

func checkIfUserIsDeletable(id string) (bool, error) {
	// No puedes borrar el usuario administrador (el que tenga el nombre de usuario del .env)
	// Se podría comprobar que el ID no sea "1" pero nunca viene mal una comprobación adicional
	adminUsername := config.LoadConfig().DefaultAdminUsername
	dbUser := db.DB.QueryRow("SELECT * FROM users WHERE id = ?", id)
	var user models.User
	err := dbUser.Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.Active)
	if err != nil {
		return false, err
	}

	return user.Username != adminUsername, nil
}

// DeleteUser elimina un usuario (desactiva el usuario)
func DeleteUser(c *fiber.Ctx) error {
	id := c.Params("user_id")
	isDeletable, err := checkIfUserIsDeletable(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al comprobar si el usuario es deletable",
		})
	}

	if !isDeletable {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "No puedes borrar el usuario administrador (el que viene en el archivo de configuración)",
		})
	}

	forceDelete := c.Query("forceDelete") == "true"
	message := "Usuario eliminado (desactivado) correctamente, si quiere eliminar el usuario, utiliza el parámetro 'forceDelete=true' en la URL, esto borrará el usuario y todos los videos convertidos de este usuario"
	// TODO: Borrar videos del usuario (almacenamiento) y utilizar una transacción para este borrado
	if forceDelete {
		_, err := db.DB.Exec("DELETE FROM videos WHERE user_id = ?", id)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al eliminar los videos del usuario",
			})
		}
		_, err = db.DB.Exec("DELETE FROM users WHERE id = ?", id)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al eliminar el usuario",
			})
		}
		message = "Usuario eliminado correctamente"
	} else {
		_, err := db.DB.Exec("UPDATE users SET active = false WHERE id = ?", id)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al eliminar el usuario",
			})
		}
	}

	return c.JSON(fiber.Map{
		"message":     message,
		"forceDelete": forceDelete,
	})
}

// GetUser obtiene un usuario
func GetUser(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	rows, err := db.DB.Query("SELECT * FROM users WHERE id = ?", userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener el usuario",
		})
	}
	defer rows.Close()

	var user models.User
	for rows.Next() {
		err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.Active)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al obtener el usuario",
			})
		}
	}

	if user.ID == "" {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Usuario no encontrado",
		})
	}

	return c.JSON(user)
}

// GetUserVideos obtiene los videos del usuario autenticado
func GetUserVideos(c *fiber.Ctx) error {
	// Obtener el token JWT del contexto
	token := c.Locals("jwt").(*jwt.Token)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Token inválido",
		})
	}

	userID := claims["user_id"].(string)
	log.Println(userID)
	// Convertir el userID de string a int
	userIDInt, err := strconv.Atoi(userID)

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al procesar el ID del usuario",
		})
	}

	// Obtener los videos del usuario
	rows, err := db.DB.Query("SELECT * FROM videos WHERE user_id = ?", userIDInt)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los videos del usuario",
		})
	}
	defer rows.Close()

	var videos []models.Video
	for rows.Next() {
		var video models.Video
		err := rows.Scan(&video.ID, &video.UserID, &video.VideoID, &video.Title, &video.RequestedByIP, &video.CreatedAt, &video.UpdatedAt)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al obtener los videos del usuario",
			})
		}
		videos = append(videos, video)
	}

	return c.JSON(fiber.Map{
		"videos": videos,
		"user":   userIDInt,
	})
}

// GetCurrentUser obtiene el usuario autenticado y sus videos convertidos
func GetCurrentUser(c *fiber.Ctx) error {
	// Obtener el token JWT del contexto
	token := c.Locals("jwt").(*jwt.Token)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Token inválido",
		})
	}

	userID := claims["user_id"].(string)

	rows, err := db.DB.Query("SELECT * FROM users WHERE id = ?", userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener el usuario",
		})
	}
	defer rows.Close()

	var user models.User
	for rows.Next() {
		err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.Active)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al obtener el usuario",
			})
		}
	}

	rows, err = db.DB.Query("SELECT * FROM videos WHERE user_id = ?", userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los videos del usuario",
		})
	}
	defer rows.Close()

	var videos []models.Video
	for rows.Next() {
		var video models.Video
		err := rows.Scan(&video.ID, &video.UserID, &video.VideoID, &video.Title, &video.RequestedByIP, &video.CreatedAt, &video.UpdatedAt)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error al obtener los videos del usuario",
			})
		}
		videos = append(videos, video)
	}

	userResponse := UserResponse{
		User:   user,
		Videos: videos,
	}

	return c.JSON(userResponse)
}

// GetVideoByUser Obtiene los videos de un usuario
func GetVideoByUser(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	rows, err := db.DB.Query("SELECT * FROM videos WHERE user_id = ?", userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los videos",
		})
	}
	defer rows.Close()

	var videos []models.Video
	for rows.Next() {
		var video models.Video
		err := rows.Scan(&video.ID, &video.UserID, &video.VideoID, &video.Title, &video.RequestedByIP, &video.CreatedAt, &video.UpdatedAt)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{})
		}
		videos = append(videos, video)
	}

	return c.JSON(videos)
}

// CreateUser crea un usuario
func CreateUser(c *fiber.Ctx) error {
	var user CreateUserRequest
	if err := c.BodyParser(&user); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Error al crear el usuario",
		})
	}

	// Comprobar si el usuario ya existe
	userExists := false
	err := db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", user.Username).Scan(&userExists)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al comprobar si el usuario existe",
		})
	}

	if userExists {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "El usuario ya existe",
		})
	}

	// Hashear contraseña y agregarlo a la base de datos
	user.Password = pkg.GeneratePassword(user.Password)

	_, err = db.DB.Exec("INSERT INTO users (username, password, role, active) VALUES (?, ?, ?, ?)", user.Username, user.Password, user.Role, user.Active)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error":      "Error al crear el usuario",
			"errorTrace": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Usuario creado correctamente",
	})
}

// UpdateUser actualiza un usuario
func UpdateUser(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	var user UpdateUserRequest
	if err := c.BodyParser(&user); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Error al actualizar el usuario",
		})
	}

	// Comprobar si el usuario existe
	userExists := false
	err := db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", userID).Scan(&userExists)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al comprobar si el usuario existe",
		})
	}

	if !userExists {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "El usuario no existe",
		})
	}

	// Comprobar si el nombre de usuario, rol y activo es actualizable (No es actualizable cuando se intenta actualizar el nombre de usuario del usuario administrador establecido por el .env, este es el usuario numero 1)
	if userID == "1" {
		user.Username = config.LoadConfig().DefaultAdminUsername
		user.Role = "admin"
		user.Active = true
	}

	// Hashear contraseña y agregarlo a la base de datos
	user.Password = pkg.GeneratePassword(user.Password)

	_, err = db.DB.Exec("UPDATE users SET username = ?, password = ?, role = ?, active = ? WHERE id = ?", user.Username, user.Password, user.Role, user.Active, userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error":      "Error al actualizar el usuario",
			"errorTrace": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Usuario actualizado correctamente",
	})
}
