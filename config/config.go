package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	JwtSecret            string
	Production           bool
	DefaultAdminUsername string
	DefaultAdminPassword string
	GoogleCloudApiKey    string
	PyConverterPath      string
	StoragePath          string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	return Config{
		Port:                 getEnv("PORT", "3000"),
		JwtSecret:            getEnv("JWT_SECRET", "super_secret_default_jwt_secret"),
		Production:           getEnv("PRODUCTION", "false") == "true",
		DefaultAdminUsername: getEnv("DEFAULT_ADMIN_USERNAME", "admin"),
		DefaultAdminPassword: getEnv("DEFAULT_ADMIN_PASSWORD", "admin"),
		GoogleCloudApiKey:    getEnv("GOOGLE_CLOUD_API_KEY", ""),
		PyConverterPath:      getEnv("PYCONVERTER_PATH", "/home/andres/Desktop/Proyectos/yt-converter-api/pkg/pyConverter/main.py"),
		StoragePath:          getEnv("STORAGE_PATH", "/home/andres/Desktop/Proyectos/yt-converter-api/storage"),
	}
}

// getEnv obtiene una variable de entorno o usa un valor por defecto

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
