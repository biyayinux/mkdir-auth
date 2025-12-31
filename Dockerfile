# [ Étape 1 ] Compilation
FROM golang:1.21-alpine AS builder

# Installation des certificats de sécurité et outils de build
RUN apk add --no-cache git ca-certificates

# Définition du répertoire de travail
WORKDIR /app

# Copie des fichiers de dépendances
COPY go.mod go.sum ./
RUN go mod download

# Copie de tout le code source
COPY . .

# Compilation du binaire (optimisé pour Docker)
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# [ Étape 2 ] Image finale
FROM alpine:latest

WORKDIR /app

# Récupération des certificats CA (pour que l'auth Google HTTPS fonctionne)
RUN apk --no-cache add ca-certificates

# Copie du binaire depuis le builder
COPY --from=builder /app/main .

# Exposition du port
EXPOSE 8080

# Lancement du serveur
CMD ["./main"]
