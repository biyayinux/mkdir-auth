package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mkdir-auth/internal/auth"
	"mkdir-auth/internal/database"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// HandleLogin initie l'authentification Google
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	// Nettoyage de l'URL d'origine
	rawOrigin := r.Header.Get("Origin")
	if rawOrigin == "" {
		rawOrigin = r.Header.Get("Referer")
	}
	cleanOrigin := strings.TrimSpace(strings.TrimSuffix(rawOrigin, "/"))
	pubKey := r.URL.Query().Get("publishable_key")

	var projet struct {
		ID      string
		Origine string
	}

	// Verification du projet et de l'origine en DB
	err := database.DB.QueryRow("SELECT id, origines_autorisees FROM projets WHERE publishable_key = $1", pubKey).
		Scan(&projet.ID, &projet.Origine)

	if err != nil {
		http.Error(w, "Projet non trouvé", http.StatusForbidden)
		return
	}

	if strings.TrimSpace(strings.TrimSuffix(projet.Origine, "/")) != cleanOrigin {
		fmt.Printf("Origine refusée. Reçu: %s, Attendu: %s\n", cleanOrigin, projet.Origine)
		http.Error(w, "Origine non autorisee", http.StatusForbidden)
		return
	}

	// Redirection vers Google avec le state (ID|URL)
	combinedState := projet.ID + "|" + cleanOrigin
	url := auth.GoogleOauthConfig.AuthCodeURL(combinedState)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// HandleCallback traite le retour de Google et crée la session
func HandleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	code := r.FormValue("code")

	// Recupération des données du state
	parts := strings.Split(state, "|")
	if len(parts) < 2 {
		http.Error(w, "State corrompu", http.StatusBadRequest)
		return
	}
	projetID, frontendURL := parts[0], parts[1]

	// Echange du code contre un token Google
	token, err := auth.GoogleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Erreur token Google", http.StatusInternalServerError)
		return
	}

	// Recupération du profil utilisateur Google
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

	// Sauvegarde ou mise à jour de l'utilisateur
	var userID string
	err = database.DB.QueryRow(`
        INSERT INTO utilisateurs (email) VALUES ($1)
        ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
        RETURNING id`, gu.Email).Scan(&userID)

	if err != nil {
		log.Println("Erreur DB utilisateurs:", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// Enregistrement de la connexion au projet
	database.DB.Exec(`
        INSERT INTO utilisateurs_projets (utilisateur_id, projet_id, derniere_connexion)
        VALUES ($1, $2, NOW())
        ON CONFLICT (utilisateur_id, projet_id) DO UPDATE SET derniere_connexion = NOW()`,
		userID, projetID)

	// Création du JWT
	claims := jwt.MapClaims{
		"uid":    userID,
		"pid":    projetID,
		"email":  gu.Email,
		"name":   gu.Name,
		"avatar": gu.Picture,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	}
	tokenString, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(os.Getenv("JWT_SECRET")))
	isSecure := os.Getenv("COOKIE_SECURE") == "true"

	// Envoi du cookie de session
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	// Retour automatique vers le frontend
	http.Redirect(w, r, frontendURL, http.StatusTemporaryRedirect)
}

// HandleMe renvoie les infos de l'utilisateur via le JWT
func HandleMe(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Content-Type", "application/json")

	cookie, err := r.Cookie("auth_token")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"authenticated": false})
		return
	}

	// Validation du token
	token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err == nil && token.Valid {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
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

// HandleLogout supprime le cookie de session
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	http.SetCookie(w, &http.Cookie{
		Name: "auth_token", Value: "", Path: "/", HttpOnly: true, MaxAge: -1,
	})
	w.WriteHeader(http.StatusOK)
}
