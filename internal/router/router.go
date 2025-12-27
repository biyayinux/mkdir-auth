package router

import (
	"mkdir-auth/internal/config"
	"mkdir-auth/internal/handlers"
	"mkdir-auth/internal/middlewares"
	"net/http"
)

func SetupRouter() *http.ServeMux {
	config.LoadEnv()
	config.InitGoogleOAuth()

	mux := http.NewServeMux()

	mux.HandleFunc("/login", handlers.Login)
	mux.HandleFunc("/callback", handlers.Callback)
	mux.HandleFunc("/logout",
		middlewares.EnableCORS(handlers.Logout),
	)
	mux.HandleFunc("/me",
		middlewares.EnableCORS(
			middlewares.AuthMiddleware(handlers.Me),
		),
	)

	return mux
}
