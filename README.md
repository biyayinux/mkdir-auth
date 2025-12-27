
# MKDIR AUTH

🔐🚀 mkdir Auth est une API d’authentification moderne en Go, conçue pour sécuriser 🔒, gérer 👥 et contrôler 🚧 les accès aux services de mkdir.
⚡☁️ Performante, scalable 📈 et cloud-native 🐹, elle fournit des tokens 🔑, une sécurité béton 🛡️ et une intégration ultra simple 🔌✨

##  Variable d'environnement

Remplacer les valeurs d’exemple par les valeurs réelles utilisées en production

```bash
  # Configuration Serveur
  SERVER_PORT=:8080
  SERVER_URL=http://localhost:8080
  FRONTEND_URL=http://localhost:5173

  # Configuration Google
  GOOGLE_CLIENT_ID=123-xxxxxx-xxxxxx.apps.googleusercontent.com
  GOOGLE_CLIENT_SECRET=xxx-xxx-xxx
  GOOGLE_REDIRECT_URL=http://localhost:8080/callback

  # Sécurité
  JWT_SECRET=xxxx-xxxx-xxxx
```

##  Récap express

```bash
  cd mkdir-auth
  go mod tidy
  go run ./cmd/server
```
