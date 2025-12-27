package handlers

import (
	"context"
	"encoding/json"
	"io"
	"mkdir-auth/internal/config"
	"mkdir-auth/internal/models"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const randomState = "secure_state_string"

func Login(w http.ResponseWriter, r *http.Request) {
	url := config.GoogleOauthConfig.AuthCodeURL(randomState)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func Callback(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("state") != randomState {
		http.Error(w, "State invalide", http.StatusBadRequest)
		return
	}

	token, err := config.GoogleOauthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		http.Error(w, "Erreur échange token", http.StatusUnauthorized)
		return
	}

	resp, err := http.Get(
		"https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken,
	)
	if err != nil {
		http.Error(w, "Erreur Google API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var gUser models.GoogleUser
	json.Unmarshal(data, &gUser)

	expiration := time.Now().Add(24 * time.Hour)

	claims := models.UserClaims{
		Email:   gUser.Email,
		Name:    gUser.Name,
		Picture: gUser.Picture,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := jwtToken.SignedString(config.JWTKey)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    tokenString,
		Expires:  expiration,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   false,
	})

	http.Redirect(w, r, config.FrontendURL, http.StatusSeeOther)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Path:     "/",
	})

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Logged out",
	})
}
