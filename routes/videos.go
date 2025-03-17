package routes

import (
	"net/http"
	"strconv"

	"yt-converter-api/db"
	"yt-converter-api/models"
	"yt-converter-api/pkg"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// GetVideos obtiene la lista de videos íntegra
func GetVideos(c *fiber.Ctx) error {
	rows, err := db.DB.Query("SELECT * FROM videos")
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los videos",
		})
	}
	defer rows.Close()

	var videos []models.Video
	for rows.Next() {
		var video models.Video
		err := rows.Scan(&video.ID, &video.UserID, &video.VideoID, &video.Title, &video.Format, &video.Path, &video.RequestedByIP, &video.CreatedAt, &video.UpdatedAt)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{})
		}
		videos = append(videos, video)
	}

	return c.JSON(videos)
}

// AddVideo agrega un nuevo video si no existe
func AddVideo(c *fiber.Ctx) error {
	type Request struct {
		URL    string `json:"url"`
		Format string `json:"format"`
	}

	var request Request
	if err := c.BodyParser(&request); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Error al analizar el cuerpo de la solicitud",
		})
	}

	// Obtener el token JWT del contexto
	token := c.Locals("jwt").(*jwt.Token)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Token inválido",
		})
	}

	userID := claims["user_id"].(string)

	// Convertir el userID de string a int
	userIDInt, err := strconv.Atoi(userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al procesar el ID del usuario",
		})
	}

	// Verificar si el video ya existe
	var exists bool
	err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM videos WHERE video_id = ? AND format = ?)", request.URL, request.Format).Scan(&exists)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al verificar si el video existe",
		})
	}

	if exists {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"converted": true,
			"message":   "El video ya existe",
			"path":      "",
		})
	}

	// Comprobar la validez de la URL
	if !pkg.IsUrl(request.URL) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "URL no válida",
		})
	}

	// Comprobar si la URL es un video de Youtube
	valid, err := pkg.IsYoutubeUrl(request.URL)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "La URL no es una URL válida de Youtube",
		})
	}

	if !valid {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "URL no es un video de Youtube",
		})
	}

	// Obtener el ID del video y el título
	videoID := pkg.GetYoutubeVideoID(request.URL)
	videoTitle, err := pkg.GetYoutubeVideoTitle(request.URL)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener el título del video",
		})
	}

	// Insertar el video en la base de datos
	video := models.Video{
		UserID:        userIDInt,
		VideoID:       pkg.GetYoutubeVideoID(request.URL),
		Title:         videoTitle,
		Format:        request.Format,
		Path:          "/storage/" + request.Format + "/" + videoID + "." + request.Format,
		RequestedByIP: c.IP(),
	}

	_, err = db.DB.Exec("INSERT INTO videos (user_id, video_id, title, format, path, requested_by_ip) VALUES (?, ?, ?, ?, ?, ?)", video.UserID, video.VideoID, video.Title, video.Format, video.Path, video.RequestedByIP)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error":      "Error al insertar el video",
			"errorTrace": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"converted": true,
		"message":   "Video agregado correctamente",
		"path":      video.Path,
	})
}

// GetVideo Obtiene los videos disponibles para un video_id
func GetVideo(c *fiber.Ctx) error {
	videoID := c.Params("video_id")
	rows, err := db.DB.Query("SELECT * FROM videos WHERE video_id = ?", videoID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al obtener los videos",
		})
	}
	defer rows.Close()

	var videos []models.Video
	for rows.Next() {
		var video models.Video
		err := rows.Scan(&video.ID, &video.UserID, &video.VideoID, &video.Title, &video.Format, &video.Path, &video.RequestedByIP, &video.CreatedAt, &video.UpdatedAt)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{})
		}
		videos = append(videos, video)
	}

	return c.JSON(videos)
}

// DeleteVideo Elimina un video de la base de datos
func DeleteVideo(c *fiber.Ctx) error {
	videoID := c.Params("video_id")
	_, err := db.DB.Exec("DELETE FROM videos WHERE video_id = ?", videoID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error al eliminar el video",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Video eliminado correctamente",
	})
}
