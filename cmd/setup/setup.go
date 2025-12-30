package main

import (
	"bufio"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Lit la saisie utilisateur
func readInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// Genere une chaine aleatoire
func generateRandomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func main() {
	godotenv.Load()

	// Connexion DB
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Erreur connexion DB:", err)
	}
	defer db.Close()

	fmt.Println("\nOUTIL DE CONFIGURATION AUTH")
	fmt.Println("---------------------------")

	// Selection de l'administrateur
	fmt.Println("\n[Etape 1] Selection de l'Administrateur")
	rows, err := db.Query("SELECT email, pseudo FROM administrateurs")
	if err != nil {
		log.Fatal("Erreur lecture admin:", err)
	}

	fmt.Println("Administrateurs disponibles :")
	for rows.Next() {
		var email, pseudo string
		rows.Scan(&email, &pseudo)
		fmt.Printf("- %s (%s)\n", pseudo, email)
	}
	rows.Close()

	adminEmail := readInput("\nEmail de l'admin : ")
	var adminID string
	err = db.QueryRow("SELECT id FROM administrateurs WHERE email = $1", adminEmail).Scan(&adminID)
	if err != nil {
		log.Fatal("Admin introuvable.")
	}

	// Details du projet
	fmt.Println("\n[Etape 2] Details du Projet")
	nomProjet := readInput("Nom du projet : ")
	uniqueOrigin := readInput("Origine autorisee (ex: http://localhost:5173) : ")

	// Generation des identifiants
	fmt.Println("\n[Etape 3] Generation des identifiants...")
	rawSecretKey := "sk_live_" + generateRandomString(24)
	publishableKey := "pk_live_" + generateRandomString(16)
	hashedSecret, _ := bcrypt.GenerateFromPassword([]byte(rawSecretKey), bcrypt.DefaultCost)

	// Utilisation de 'origines_autorisees' pour correspondre a la table
	query := `
        INSERT INTO projets (nom, publishable_key, secret_key_hash, origines_autorisees, administrateur_id)
        VALUES ($1, $2, $3, $4, $5)`

	_, err = db.Exec(query, nomProjet, publishableKey, string(hashedSecret), uniqueOrigin, adminID)
	if err != nil {
		log.Fatal("Erreur SQL insertion :", err)
	}

	// Recapitulatif
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("PROJET CREE AVEC SUCCES")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Projet        : %s\n", nomProjet)
	fmt.Printf("Origine       : %s\n", uniqueOrigin)
	fmt.Printf("Public Key    : %s\n", publishableKey)
	fmt.Printf("Secret Key    : %s\n", rawSecretKey)
	fmt.Println(strings.Repeat("=", 60))
}
