package main

import (
	"fmt"
	"log"
	"mkdir-auth/internal/auth"
	"mkdir-auth/internal/database"
	"mkdir-auth/internal/transport"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	// Initialisation
	database.InitDB()
	serverURL := os.Getenv("SERVER_URL")
	port := os.Getenv("PORT")
	auth.InitGoogleOAuth(serverURL)

	// Routes
	http.HandleFunc("/login", transport.HandleLogin)
	http.HandleFunc("/callback", transport.HandleCallback)
	http.HandleFunc("/me", transport.HandleMe)
	http.HandleFunc("/logout", transport.HandleLogout)

	fmt.Println("Serveur Auth Multi-Locataire sur " + serverURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
