package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	JWTKey      []byte
	FrontendURL string
)

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: Fichier .env non chargé")
	}

	JWTKey = []byte(os.Getenv("JWT_SECRET"))
	FrontendURL = os.Getenv("FRONTEND_URL")
}
