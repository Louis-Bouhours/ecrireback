# Ecrireback API

API backend pour l'application de chat Ecrire.

## Endpoints

### Authentification

#### POST /login
Connexion utilisateur directe (legacy)

**Body:**
```json
{
  "username": "string",
  "password": "string"
}
```

#### POST /api/login
Connexion utilisateur via API

**Body:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Response:**
```json
{
  "username": "string",
  "avatar": "string"
}
```

#### POST /logout
Déconnexion utilisateur (legacy)

#### POST /api/logout
Déconnexion utilisateur via API

**Response:**
```json
{
  "message": "Déconnexion réussie"
}
```

#### POST /api/register
Inscription d'un nouvel utilisateur

**Body:**
```json
{
  "username": "string",
  "password": "string",
  "avatar": "string"
}
```

**Response:**
```json
{
  "username": "string",
  "avatar": "string"
}
```

### Chat

#### Socket.io: /socket.io/*
Gestion des connexions WebSocket pour le chat temps réel.

**Events:**
- `join`: Rejoindre le salon avec un token
- `message`: Envoyer un message
- `user_list`: Liste des utilisateurs connectés

### Autres

#### GET /health
Check de santé du serveur

**Response:**
```json
{
  "message": "YOUPY"
}
```

#### GET /profile
Profil utilisateur (nécessite authentification)

**Response:**
```json
{
  "message": "Bienvenue {username}"
}
```

## Authentification

L'API utilise JWT (JSON Web Tokens) pour l'authentification. Les tokens sont stockés dans des cookies HttpOnly.

## Configuration

Voir le fichier `configs/.env` pour la configuration des variables d'environnement.