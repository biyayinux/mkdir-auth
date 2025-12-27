package main

import (
	"log"
	"mkdir-auth/internal/router"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = ":8080"
	}

	mux := router.SetupRouter()

	log.Printf("Serveur démarré sur %s\n", port)
	log.Fatal(http.ListenAndServe(port, mux))
}
