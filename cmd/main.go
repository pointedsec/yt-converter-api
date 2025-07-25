package main

import (
	"log"
	"os"
	"yt-converter-api/config"
	"yt-converter-api/db"
	"yt-converter-api/middleware"
	"yt-converter-api/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	cfg := config.LoadConfig()
	// Si no se ha establecido la API Key de Google Cloud, se muestra un error y la aplicación se cierra
	if cfg.GoogleCloudApiKey == "" {
		log.Fatal("Google Cloud API Key not set")
		os.Exit(1)
	}

	// Iniciar la aplicación y configurar CORS
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowHeaders: "Origin,Content-Type,Accept,Content-Length,Accept-Language,Accept-Encoding,Connection,Access-Control-Allow-Origin,Authorization",
		AllowOrigins: "*",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
	}))

	// Iniciar la base de datos
	db.InitDB()

	api := app.Group("/api")

	// Status
	api.Get("/status", routes.GetStatus)

	// Rutas
	/* -----------------------------------------------------------------
	|                                                                   |
	|                             VIDEOS                                |
	|                                                                   |
	------------------------------------------------------------------- */
	videos := api.Group("/videos")
	videos.Use(middleware.JWTProtected())
	videos.Use(middleware.ValidUserAndActive)

	// ADMIN
	videos.Get("/", middleware.IsAdmin, routes.GetVideos)               // Obtiene todos los videos
	videos.Delete("/:video_id", middleware.IsAdmin, routes.DeleteVideo) // Elimina un video
	// Usuarios
	videos.Post("/", routes.AddVideo)                       // Inserta un video
	videos.Get("/:video_id", routes.GetVideo)               // Obtiene un video de la BBDD
	videos.Get(":video_id/formats", routes.GetVideoFormats) // Obtiene los formatos disponibles de un video (resoluciones)
	videos.Post(":video_id/process", routes.ProcessVideo)   // Procesa un video con el formato (resolución) indicado por POST, es decir, descarga el video y lo almacena en su correspondiente carpeta
	videos.Get(":video_id/download", routes.DownloadVideo)  // Descarga un video
	videos.Get(":video_id/status", routes.GetVideoStatus)   // Obtiene el estado de procesamiento de un video

	/* -----------------------------------------------------------------
	|                                                                   |
	|                             USERS                                |
	|                                                                   |
	------------------------------------------------------------------- */
	users := api.Group("/users")
	users.Use(middleware.JWTProtected())
	users.Use(middleware.ValidUserAndActive)

	// Usuarios
	users.Get("/me", routes.GetCurrentUser) // Obtiene el usuario autenticado y sus videos convertidos
	// ADMIN
	users.Post("/", middleware.IsAdmin, routes.CreateUser)                   // Crea un usuario
	users.Put("/:user_id", middleware.IsAdmin, routes.UpdateUser)            // Actualiza un usuario
	users.Get("/", middleware.IsAdmin, routes.GetUsers)                      // Obtiene todos los usuarios
	users.Delete("/:user_id", middleware.IsAdmin, routes.DeleteUser)         // Elimina un usuario
	users.Get("/:user_id", middleware.IsAdmin, routes.GetUser)               // Obtiene un usuario
	users.Get("/:user_id/videos", middleware.IsAdmin, routes.GetVideoByUser) // Obtiene los videos de un usuario

	/* -----------------------------------------------------------------
	|                                                                   |
	|                             COOKIES                               |
	|                                                                   |
	------------------------------------------------------------------- */
	cookies := api.Group("/cookies")
	cookies.Use(middleware.JWTProtected())
	cookies.Use(middleware.ValidUserAndActive)

	// ADMIN
	cookies.Post("/", middleware.IsAdmin, routes.UploadCookies)       // Subir un archivo cookies.txt para usarlo con yt-dlp
	cookies.Get("/", middleware.IsAdmin, routes.GetCookiesInfo)       // Comprobar si existe ya un archivo cookies.txt
	cookies.Delete("/", middleware.IsAdmin, routes.DeleteCookiesFile) // Borrar el archivo de cookies si ya existe

	/* -----------------------------------------------------------------
	|                                                                   |
	|                             AUTH                                  |
	|                                                                   |
	------------------------------------------------------------------- */
	auth := api.Group("/auth")
	auth.Post("/login", routes.Login)
	auth.Get("/logout", routes.Logout)

	port := cfg.Port
	log.Printf("Server is running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
