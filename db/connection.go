package db

import (
	"database/sql"
	"log"
	"yt-converter-api/config"
	"yt-converter-api/pkg"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite3", "db/database.db")
	if err != nil {
		log.Fatal("Error abriendo la base de datos:", err)
	}
	if !config.LoadConfig().Production {
		log.Println("Modo desarrollo activado, eliminando tablas y creando nuevas")
		deleteTables()
	}
	createTables()
	if !config.LoadConfig().Production {
		log.Println("Creando administrador por defecto, credenciales: ", config.LoadConfig().DefaultAdminUsername, config.LoadConfig().DefaultAdminPassword)
		createDefaultAdmin()
	}
}

func deleteTables() {
	query := `
	DROP TABLE IF EXISTS users;
	DROP TABLE IF EXISTS videos;
	DROP TABLE IF EXISTS video_status;
	`
	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal("Error eliminando tablas:", err)
	}
}

func createTables() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		role TEXT CHECK(role IN ('admin', 'guest')) NOT NULL,
		active BOOLEAN DEFAULT TRUE
	);
	CREATE TABLE IF NOT EXISTS videos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		video_id TEXT NOT NULL,
		title TEXT NOT NULL,
		requested_by_ip TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(user_id) REFERENCES users(id),
		UNIQUE(video_id)
	);
	CREATE TABLE IF NOT EXISTS video_status (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		video_id TEXT NOT NULL,
		resolution TEXT NOT NULL,
		path TEXT,
		status TEXT CHECK(status IN ('processing', 'completed', 'failed')) NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(video_id) REFERENCES videos(video_id),
		UNIQUE(video_id, resolution)
	);
	`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatal("Error creando tablas:", err)
	}
}

func createDefaultAdmin() {
	query := `
	INSERT INTO users (username, password, role) VALUES (?, ?, ?)`
	_, err := DB.Exec(query, config.LoadConfig().DefaultAdminUsername, pkg.GeneratePassword(config.LoadConfig().DefaultAdminPassword), "admin")
	if err != nil {
		log.Fatal("Error creando administrador por defecto:", err)
	}
}
