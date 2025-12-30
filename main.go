package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOauthConfig *oauth2.Config
	db                *sql.DB
)

func init() {
	godotenv.Load()
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Erreur connexion DB:", err)
	}

	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/callback",
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
}

func main() {
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/callback", handleCallback)
	http.HandleFunc("/me", handleMe)
	http.HandleFunc("/logout", handleLogout)

	fmt.Println("Serveur Auth Multi-Tenant sur http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	// Gestion CORS
	origin := r.Header.Get("Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	// Nettoyage de l'origine
	rawOrigin := r.Header.Get("Origin")
	if rawOrigin == "" {
		rawOrigin = r.Header.Get("Referer")
	}
	cleanOrigin := strings.TrimSpace(strings.TrimSuffix(rawOrigin, "/"))

	pubKey := r.URL.Query().Get("publishable_key")

	var projet struct {
		ID      string
		Origine string // Maintenant une simple string
	}

	// Récupération de l'origine unique depuis la DB
	err := db.QueryRow("SELECT id, origines_autorisees FROM projets WHERE publishable_key = $1", pubKey).
		Scan(&projet.ID, &projet.Origine)

	if err != nil {
		http.Error(w, "Projet non trouvé", http.StatusForbidden)
		return
	}

	// Comparaison stricte (Une seule origine autorisée)
	if strings.TrimSpace(strings.TrimSuffix(projet.Origine, "/")) != cleanOrigin {
		fmt.Printf("Origine refusée. Reçu: %s, Attendu: %s\n", cleanOrigin, projet.Origine)
		http.Error(w, "Origine non autorisee", http.StatusForbidden)
		return
	}

	// State combiné
	combinedState := projet.ID + "|" + cleanOrigin
	url := googleOauthConfig.AuthCodeURL(combinedState)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	code := r.FormValue("code")

	// Découpage du state dynamique
	parts := strings.Split(state, "|")
	if len(parts) < 2 {
		http.Error(w, "State corrompu", http.StatusBadRequest)
		return
	}
	projetID := parts[0]
	frontendURL := parts[1]

	// Échange token Google
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Erreur token Google", http.StatusInternalServerError)
		return
	}

	// Infos utilisateur Google
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		http.Error(w, "Erreur infos Google", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var gu struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	json.NewDecoder(resp.Body).Decode(&gu)

	// Insertion Email (Table utilisateurs)
	var userID string
	err = db.QueryRow(`
		INSERT INTO utilisateurs (email) VALUES ($1)
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING id`, gu.Email).Scan(&userID)

	if err != nil {
		log.Println("Erreur DB utilisateurs:", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// Liaison Projet (Table utilisateurs_projets)
	_, err = db.Exec(`
		INSERT INTO utilisateurs_projets (utilisateur_id, projet_id, derniere_connexion)
		VALUES ($1, $2, NOW())
		ON CONFLICT (utilisateur_id, projet_id) DO UPDATE SET derniere_connexion = NOW()`,
		userID, projetID)

	// Génération JWT avec données profil (Nom/Avatar)
	claims := jwt.MapClaims{
		"uid":    userID,
		"pid":    projetID,
		"email":  gu.Email,
		"name":   gu.Name,
		"avatar": gu.Picture,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	}
	tokenString, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(os.Getenv("JWT_SECRET")))

	// Cookie sécurisé
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Mettre à true en production (HTTPS)
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	// Retour dynamique vers l'origine précise
	http.Redirect(w, r, frontendURL, http.StatusTemporaryRedirect)
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Content-Type", "application/json")

	cookie, err := r.Cookie("auth_token")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"authenticated": false})
		return
	}

	token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err == nil && token.Valid {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// JWT vitesse maximale, pas de SQL
			user := map[string]interface{}{
				"id":     claims["uid"],
				"email":  claims["email"],
				"name":   claims["name"],
				"avatar": claims["avatar"],
			}
			json.NewEncoder(w).Encode(user)
			return
		}
	}

	w.WriteHeader(http.StatusUnauthorized)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	http.SetCookie(w, &http.Cookie{
		Name: "auth_token", Value: "", Path: "/", HttpOnly: true, MaxAge: -1,
	})
	w.WriteHeader(http.StatusOK)
}
