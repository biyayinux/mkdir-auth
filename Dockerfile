# [ Étape 1 ] Compilation
FROM golang:1.25-alpine AS builder

# Installation des outils nécessaires
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Gestion des dépendances (mise en cache)
COPY go.mod go.sum ./
RUN go mod download

# Copie de l'intégralité du projet
COPY . .

# Compilation du binaire en pointant vers le bon dossier cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-server ./cmd/server/main.go

# [ Étape 2 ] Image finale légère
FROM alpine:latest

# Installation des certificats pour les appels API Google (HTTPS)
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Récupération du binaire compilé
COPY --from=builder /app/auth-server .

# Le port est exposé dynamiquement via la variable d'env PORT, 
# mais on indique 8080 par défaut ici.
EXPOSE 8080

# Lancement de l'application
CMD ["./auth-server"]
