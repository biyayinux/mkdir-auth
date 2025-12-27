package handlers

import (
	"encoding/json"
	"mkdir-auth/internal/models"
	"net/http"
)

func Me(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(*models.UserClaims)

	if !ok {
		json.NewEncoder(w).Encode(map[string]bool{
			"authenticated": false,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"email":         user.Email,
		"name":          user.Name,
		"picture":       user.Picture,
	})
}
