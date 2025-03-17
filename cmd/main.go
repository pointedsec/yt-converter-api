package main

import (
	"log"
	"os"
	"yt-converter-api/config"
	"yt-converter-api/db"
	"yt-converter-api/middleware"
	"yt-converter-api/routes"

	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg := config.LoadConfig()
	// Si no se ha establecido la API Key de Google Cloud, se muestra un error y la aplicación se cierra
	if cfg.GoogleCloudApiKey == "" {
		log.Fatal("Google Cloud API Key not set")
		os.Exit(1)
	}

	// Iniciar la aplicación y la BBDD
	app := fiber.New()
	db.InitDB()

	api := app.Group("/api")

	// Rutas
	/* -----------------------------------------------------------------
	|                                                                   |
	|                             VIDEOS                                |
	|                                                                   |
	------------------------------------------------------------------- */
	videos := api.Group("/videos")
	videos.Use(middleware.JWTProtected)

	// ADMIN
	videos.Get("/", middleware.IsAdmin, routes.GetVideos)               // Obtiene todos los videos
	videos.Delete("/:video_id", middleware.IsAdmin, routes.DeleteVideo) // Elimina un video
	// Usuarios
	videos.Post("/", routes.AddVideo)         // Inserta un video
	videos.Get("/:video_id", routes.GetVideo) // Obtiene un video y sus formatos disponibles
	// videos.GET(":video_id/formats", routes.GetVideoFormats) // Obtiene los formatos disponibles de un video (resoluciones)
	// videos.POST(":video_id/process", routes.ProcessVideo) // Procesa un video con el formato (resolución) indicado por POST, es decir, descarga el video y lo almacena en su correspondiente carpeta
	// videos.GET(":video_id/download", routes.DownloadVideo) // Descarga un video
	/* -----------------------------------------------------------------
	|                                                                   |
	|                             USERS                                |
	|                                                                   |
	------------------------------------------------------------------- */
	users := api.Group("/users")
	users.Use(middleware.JWTProtected)
	// ADMIN
	users.Post("/", middleware.IsAdmin, routes.CreateUser)                   // Crea un usuario
	users.Put("/:user_id", middleware.IsAdmin, routes.UpdateUser)            // Actualiza un usuario
	users.Get("/", middleware.IsAdmin, routes.GetUsers)                      // Obtiene todos los usuarios
	users.Delete("/:user_id", middleware.IsAdmin, routes.DeleteUser)         // Elimina un usuario
	users.Get("/:user_id", middleware.IsAdmin, routes.GetUser)               // Obtiene un usuario
	users.Get("/:user_id/videos", middleware.IsAdmin, routes.GetVideoByUser) // Obtiene los videos de un usuario
	// Usuarios
	users.Get("/me", routes.GetCurrentUser) // Obtiene el usuario autenticado y sus videos convertidos

	/* -----------------------------------------------------------------
	|                                                                   |
	|                             AUTH                                  |
	|                                                                   |
	------------------------------------------------------------------- */
	auth := api.Group("/auth")
	auth.Post("/login", routes.Login)

	port := cfg.Port
	log.Printf("Server is running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
