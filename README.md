# Ecrireback

API backend pour l'application de chat en temps réel Ecrire.

## 🏗️ Structure du projet

Le projet suit les conventions Go standards pour une organisation claire et maintenable :

```
ecrireback/
├── cmd/
│   └── server/
│       └── main.go                 # Point d'entrée principal
├── internal/
│   ├── api/
│   │   └── handlers/               # Gestionnaires HTTP
│   ├── auth/                       # Module d'authentification
│   ├── chat/                       # Module de chat WebSocket
│   ├── models/                     # Modèles de données
│   ├── config/                     # Configuration de l'application
│   └── database/                   # Connexion base de données
├── pkg/
│   └── utils/                      # Utilitaires partagés
├── configs/
│   └── .env                        # Variables d'environnement
├── scripts/
│   ├── build.sh                    # Script de build
│   └── deploy.sh                   # Script de déploiement
├── docs/
│   └── api.md                      # Documentation API
├── build/                          # Dossier pour les fichiers compilés
├── web/                            # Partie front-end/Node.js
│   ├── package.json
│   ├── package-lock.json
│   └── server.js
├── go.mod
├── go.sum
└── README.md
```

## 🚀 Installation et démarrage

### Prérequis

- Go 1.21+
- MongoDB
- Redis

### Configuration

1. Cloner le repository :
```bash
git clone https://github.com/Louis-Bouhours/ecrireback.git
cd ecrireback
```

2. Configurer les variables d'environnement dans `configs/.env` :
```env
JWT_SECRET=your_jwt_secret_here
MONGO_URI=mongodb://localhost:27017/ecriredb?authSource=admin
REDIS_ADDR=localhost:6379
SERVER_PORT=:8080
ALLOWED_ORIGINS=http://localhost:3000
```

3. Installer les dépendances :
```bash
go mod download
```

### Démarrage rapide

**Option 1: Script de déploiement**
```bash
./scripts/deploy.sh
```

**Option 2: Build et exécution manuelle**
```bash
# Build
./scripts/build.sh

# Démarrer
./build/ecrireback
```

**Option 3: Développement**
```bash
go run ./cmd/server
```

## 🔧 Développement

### Structure des modules

- **`internal/api/handlers/`** : Gestionnaires HTTP pour les endpoints API
- **`internal/auth/`** : Authentification JWT et middleware
- **`internal/chat/`** : WebSocket et gestion du chat temps réel
- **`internal/models/`** : Modèles de données MongoDB
- **`internal/config/`** : Configuration centralisée
- **`internal/database/`** : Connexion et gestion de la base de données
- **`pkg/utils/`** : Utilitaires partagés (validation, réponses HTTP)

### Build

```bash
# Build pour la plateforme actuelle
go build -o build/ecrireback ./cmd/server

# Ou utiliser le script
./scripts/build.sh
```

### Tests

```bash
go test ./...
```

## 📡 API

Voir [docs/api.md](docs/api.md) pour la documentation complète de l'API.

### Endpoints principaux

- `POST /api/login` - Connexion utilisateur
- `POST /api/register` - Inscription utilisateur
- `POST /api/logout` - Déconnexion
- `GET /health` - Check de santé
- `GET /profile` - Profil utilisateur (authentifié)
- `WebSocket /socket.io/*` - Chat temps réel

## 🛠️ Technologies utilisées

- **Go** - Langage principal
- **Gin** - Framework web
- **MongoDB** - Base de données
- **Redis** - Cache et sessions
- **Socket.IO** - WebSocket pour le chat
- **JWT** - Authentification
- **Docker** (optionnel) - Conteneurisation

## 🌐 Frontend

Le code frontend Node.js se trouve dans le dossier `web/`.

## 🔒 Sécurité

- Authentification JWT avec tokens HttpOnly
- Middleware de validation
- Gestion des sessions Redis
- CORS configuré

## 📝 Licence

Ce projet est sous licence MIT.

## 🤝 Contribution

Les contributions sont les bienvenues ! Merci de suivre les conventions Go et de maintenir la structure du projet.

## 👥 Auteurs

- Louis Bouhours - [@Louis-Bouhours](https://github.com/Louis-Bouhours)