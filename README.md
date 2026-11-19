# Bingo

Application de bingo auto-hébergée : on crée des grilles personnalisées (3×3 à 6×6), on valide
les cases au fil du temps, et chaque validation est enregistrée immédiatement dans SQLite avec
son historique.

- **Backend** : Go (bibliothèque standard) + SQLite via `modernc.org/sqlite` (pur Go, sans CGO).
- **Front** : Vue 3 + Vuetify 4, compilé par Vite et **embarqué dans le binaire Go**.
- **Déploiement** : un seul binaire, un seul fichier de base. Pas d'authentification.

## Fonctionnalités

- Plusieurs bingos en parallèle, avec progression et aperçu de la grille sur l'accueil.
- Création d'une grille : titre, description, taille, cases saisies une à une ou collées en liste,
  mélange aléatoire.
- Validation d'une case (avec note facultative) : sauvegardée à l'instant où elle est validée.
  Une validation peut être annulée.
- Détection des lignes, colonnes et diagonales complètes (« Bingo ! ») et de la grille complète.
- Historique par bingo et historique global (validations, annulations, lignes complétées…).
- Archivage (lecture seule), « Rejouer » (copie vierge d'une grille, mélangée ou non), suppression.
- Thème clair / sombre, utilisable sur mobile, aucune ressource chargée depuis Internet.

## Prérequis

- Go ≥ 1.26
- Node.js ≥ 22 et npm (uniquement pour compiler le front)

## Compiler et lancer

```bash
make build   # compile le front (web/dist) puis le binaire ./bingo
./bingo      # http://localhost:8080
```

Le front étant embarqué à la compilation, il faut toujours compiler le front **avant** le binaire
Go ; `make build` s'en charge.

### Configuration

| Option  | Variable d'env. | Défaut     | Rôle                                                   |
| ------- | --------------- | ---------- | ------------------------------------------------------ |
| `-addr` | `BINGO_ADDR`    | `:8080`    | Adresse d'écoute (`127.0.0.1:8080` pour rester en local) |
| `-db`   | `BINGO_DB`      | `bingo.db` | Chemin du fichier SQLite (créé au premier lancement)   |

L'option en ligne de commande l'emporte sur la variable d'environnement.

> L'application n'a aucune authentification : quiconque atteint le port peut tout modifier.
> À réserver au réseau local, ou à placer derrière un reverse proxy qui gère l'accès.

## Développement

Deux terminaux :

```bash
go run . -addr 127.0.0.1:8080 -db bingo-dev.db   # API
```

```bash
cd web && npm install && npm run dev             # front avec rechargement à chaud (port 5173)
```

Le serveur Vite relaie `/api` vers `localhost:8080`. `go run .` nécessite que `web/dist` existe
(lancer `make web` une première fois).

```bash
make test    # go vet + tests Go + vérification des types TypeScript
```

## Déploiement avec systemd

Un exemple d'unité est fourni dans [deploy/bingo.service](deploy/bingo.service).

```bash
make linux   # build/bingo-linux-amd64 et build/bingo-linux-arm64
```

Sur le serveur (copier le binaire adapté et le fichier `.service`) :

```bash
sudo install -m 755 bingo-linux-amd64 /usr/local/bin/bingo
sudo install -m 644 bingo.service /etc/systemd/system/bingo.service
sudo systemctl daemon-reload
sudo systemctl enable --now bingo
journalctl -u bingo -f
```

L'unité utilise `DynamicUser` et `StateDirectory` : systemd crée un utilisateur dédié et le
dossier `/var/lib/bingo`, où se trouve la base. L'adresse d'écoute se règle avec
`Environment=BINGO_ADDR=…` dans l'unité.

### Sauvegarde

Toutes les données sont dans le fichier SQLite (mode WAL). Pour une copie cohérente à chaud :

```bash
sudo sqlite3 /var/lib/bingo/bingo.db ".backup '/chemin/sauvegarde/bingo.db'"
```

Ou arrêter le service et copier `bingo.db`.

## API

Toutes les routes sont sous `/api` et échangent du JSON. Les erreurs ont la forme
`{"error": "message"}`.

| Méthode  | Route                                     | Rôle                                         |
| -------- | ----------------------------------------- | -------------------------------------------- |
| `GET`    | `/bingos`                                 | Liste des bingos (avec progression)          |
| `POST`   | `/bingos`                                 | Crée un bingo `{title, description, size, cells[]}` |
| `GET`    | `/bingos/{id}`                            | Détail d'un bingo et de ses cases            |
| `PUT`    | `/bingos/{id}`                            | Modifie titre, description et libellés       |
| `DELETE` | `/bingos/{id}`                            | Supprime le bingo et son historique          |
| `POST`   | `/bingos/{id}/archive`                    | Archive (lecture seule)                      |
| `DELETE` | `/bingos/{id}/archive`                    | Désarchive                                   |
| `POST`   | `/bingos/{id}/duplicate`                  | Copie vierge `{title, shuffle}`              |
| `PUT`    | `/bingos/{id}/cells/{cellId}/validation`  | Valide une case `{note}`                     |
| `DELETE` | `/bingos/{id}/cells/{cellId}/validation`  | Annule la validation                         |
| `GET`    | `/bingos/{id}/history`                    | Historique d'un bingo                        |
| `GET`    | `/history`                                | Historique global (`?limit=`, 1000 max)      |

## Structure

```
main.go               point d'entrée : options, serveur HTTP, arrêt propre
internal/store/       SQLite : schéma, migrations, règles métier, tests
internal/api/         routes JSON + service du front embarqué (SPA)
web/                  front Vue 3 + Vuetify (web/dist est embarqué par web/embed.go)
deploy/bingo.service  exemple d'unité systemd
```
