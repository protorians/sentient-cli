# Sentient CLI (`sentients`)

> **Statut : PLANNING (spécification complète, aucun code)**
>
> Ce document est la **spécification SpecKit de la CLI `sentients`**, outil en ligne de commande
> permettant aux développeurs d'initialiser, créer, construire, auditer, déboguer et publier des
> modules Sentient via un compte développeur `sentient-connect`.
>
> - **Stack technique** : Go (GoLang) + Bubbletea (TUI framework lipgloss/charmbracelet)
> - **Distribution** : binaire unique multi-plateforme (Linux, macOS, Windows)
> - **Aucun code n'est implémenté** à partir de ce document tant que la roadmap n'est pas engagée.

---

## Métadonnées SpecKit

| Propriété | Valeur |
|-----------|--------|
| Identifiant | `sentients` |
| Nom | Sentient CLI |
| Rôle | Outil CLI pour le cycle de vie complet des modules Sentient |
| Type de spécification | Application Spec |
| Version de spécification | `0.0.0` (candidate `0.1.0`) |
| Statut de la version | `active` (spec) — non livrée |
| Langue | Français |
| Emplacement cible (SpecKit) | `sentient.md` |

---

## 1. Vision Produit

### 1.1 Objectif

Sentient CLI est l'outil de développement unique pour tout développeur souhaitant créer, maintenir
et publier des modules dans l'écosystème Sentient. Elle couvre le cycle de vie complet :

```
init → create → develop → debug → audit → pack → sign → link → publish
                ↑                                               │
                └───────────────────────────────────────────────┘
```

### 1.2 Public cible

| Acteur | Usage |
|--------|-------|
| Développeur Sentient | Initialiser un projet, créer/modifier des modules, publier sur le store |
| Équipe interne | Audit automatique, validation des conventions, debug |

### 1.3 Valeur ajoutée

- **Zéro configuration manuelle** : détection automatique des outils disponibles sur la machine
- **Sécurité native** : credentials chiffrés, MFA supportée, token rotation
- **Validation continue** : audit des règles Clean Architecture + conformité manifest
- **Intégration Sentient Connect** : publication one-shot vers le store

---

## 2. Portée

### Dans le périmètre (In Scope)

- `sentients init` — Initialisation d'un projet Sentient (clone + deps)
- `sentients create module` — Création de module dans `external_modules/`
- `sentients connect` — Authentification développeur (credentials + MFA)
- `sentients disconnect` — Suppression des credentials
- `sentients pack` — Build + compression d'un module (`.smp`)
- `sentients sign` — Signature numérique Ed25519 des archives `.smp` (keygen / sign / verify)
- `sentients publish` — Publication dans le store via Sentient Connect
- `sentients link` — Liaison module local ↔ module en ligne
- `sentients unlink` — Dé liaison module local ↔ module en ligne
- `sentients debug <module>` — Debug d'un ou tous les modules
- `sentients audit <module>` — Audit de conformité d'un ou tous les modules
- `sentients help` — Affichage de l'aide
- `sentients -v | --version` — Affichage de la version

### Hors périmètre (Out of Scope)

- Gestion du contenu des modules (pages, composants, API)
- Monitoring temps réel des modules en production
- Gestion des organisations / équipes
- CI/CD intégré (workflow GitHub Actions séparé)

### Périmètre futur (Future Scope)

- `sentients test <module>` — Exécution des tests d'un module
- `sentients watch` — Mode développement hot-reload
- `sentients deploy` — Déploiement direct vers un environnement
- `sentients auth` — Authentification OAuth2 PKCE (navigation navigateur)
- `sentients marketplace` — Recherche/installation de modules tiers

---

## 3. Exigences

### Exigences fonctionnelles

| ID | Description |
|----|-------------|
| FR-001 | La CLI détecte automatiquement les gestionnaires de paquets disponibles (bun, pnpm, yarn, npm) et propose le choix à l'utilisateur |
| FR-002 | `sentients init` clone le repository `protorians/sentient-cms` dans le répertoire courant |
| FR-003 | `sentients init` installe les dépendances avec le gestionnaire choisi |
| FR-004 | `sentients create module` crée un module dans `external_modules/<nom>/` avec structure standardisée |
| FR-005 | `sentients create module` génère un token UUID unique dans `manifest.json` |
| FR-006 | `sentients connect` authentifie le développeur via `sentient-connect` (email + mot de passe) |
| FR-007 | `sentients connect` supporte le MFA (TOTP, backup codes) |
| FR-008 | `sentients connect` stocke les credentials de manière sécurisée (keychain/credential store) |
| FR-009 | `sentients disconnect` supprime toutes les credentials stockées |
| FR-010 | `sentients pack` compresse `external_modules/<module>/` + `public/assets/<module>/` en `.smp` |
| FR-011 | `sentients pack` déplace l'archive vers `.sentients/build/` |
| FR-012 | `sentients publish` construit puis publie via l'API `sentient-connect` |
| FR-013 | `sentients publish` demande les métadonnées du module si non définies |
| FR-014 | `sentients link` lie un module local à un module existant dans `sentient-connect` |
| FR-015 | `sentients unlink` délie un module local de `sentient-connect` |
| FR-016 | `sentients debug` lance le debug d'un module ou de tous les modules |
| FR-017 | `sentients audit` vérifie la conformité Clean Architecture, `manifest.json` et `index.tsx` |
| FR-018 | `sentients audit` vérifie que les `requirements` et `dependencies` existent |
| FR-019 | `sentients help` affiche l'aide contextuelle des commandes |
| FR-020 | `sentients -v` / `sentients --version` affiche la version actuelle |
| FR-021 | `sentients sign keygen` génère une paire de clés Ed25519 et la stocke dans le keychain système |
| FR-022 | `sentients sign <module>` signe l'archive `.smp` du module et produit un fichier `.sig` |
| FR-023 | `sentients sign verify <module>` vérifie la validité de la signature `.sig` d'un module |
| FR-024 | `sentients sign` affiche le fingerprint SHA-256 de la clé publique du développeur |

### Exigences non-fonctionnelles

| ID | Description |
|----|-------------|
| NFR-001 | Binaire unique, sans dépendance externe (static linking) |
| NFR-002 | Temps de démarrage < 100ms |
| NFR-003 | Compatible Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64) |
| NFR-004 | Sortie terminal compatible UTF-8 + 256 couleurs minimum |
| NFR-005 | Logs activables via `--verbose` ou variable d'environnement `SENTIENT_CLI_DEBUG` |
| NFR-006 | Mises à jour auto-detectées (notification, pas de mise à jour forcée) |

### Exigences de sécurité

| ID | Description |
|----|-------------|
| SEC-001 | Credentials stockés dans le keychain système (macOS Keychain, Linux secret-service, Windows Credential Manager) |
| SEC-002 | Jamais de credentials en clair sur disque (pas de `.env`, pas de fichier texte) |
| SEC-003 | Tokens d'accès avec durée de vie limitée, rotation automatique |
| SEC-004 | Chiffrement des données sensibles au repos (AES-256-GCM pour les caches) |
| SEC-005 | Validation stricte des inputs (UUID, noms de module, URLs) |
| SEC-006 | Mode MFA obligatoire si activé sur le compte développeur |
| SEC-007 | Les archives `.smp` ne contiennent jamais de credentials ou tokens |
| SEC-008 | Les clés de signature Ed25519 sont stockées dans le keychain OS, jamais en clair sur disque |
| SEC-009 | La signature numérique garantit l'intégrité et l'authenticité des archives `.smp` avant publication |

### Exigences techniques

| ID | Description |
|----|-------------|
| TECH-001 | Go 1.22+ comme langage de développement |
| TECH-002 | Bubbletea comme framework TUI pour les interactions utilisateur |
| TECH-003 | Lipgloss pour le styling terminal |
| TECH-004 | Bubbles pour les composants TUI réutilisables (spinners, selects, inputs) |
| TECH-005 | GoReleaser pour la compilation multi-plateforme et le packaging |
| TECH-006 | Architecture en couches : commands → services → infrastructure |
| TECH-007 | Configuration via fichier `.sentient-cli.toml` optionnel dans le projet |
| TECH-008 | Communication avec `sentient-connect` via REST API HTTPS |

---

## 4. Architecture

### 4.1 Architecture en couches

```
sentient-cli/
├── main.go                        # Point d'entrée
├── cmd/                           # Commandes CLI (couche présentation)
│   ├── root.go                    # Commande racine (cobra/flag parsing)
│   ├── init.go                    # sentients init
│   ├── create.go                  # sentients create module
│   ├── connect.go                 # sentients connect
│   ├── disconnect.go              # sentients disconnect
│   ├── pack.go                    # sentients pack
│   ├── sign.go                    # sentients sign (keygen / sign / verify)
│   ├── publish.go                 # sentients publish
│   ├── link.go                    # sentients link
│   ├── unlink.go                  # sentients unlink
│   ├── debug.go                   # sentients debug
│   ├── audit.go                   # sentients audit
│   ├── help.go                    # sentients help
│   └── version.go                 # sentients -v / --version
├── internal/
│   ├── config/                    # Configuration projet & CLI
│   │   ├── config.go              # Lecture/écriture .sentient-cli.toml
│   │   └── paths.go               # Résolution des chemins projet
│   ├── auth/                      # Authentification & credentials
│   │   ├── credentials.go         # Gestion keychain (CRUD)
│   │   ├── connector.go           # Client API sentient-connect
│   │   ├── mfa.go                 # Logique MFA (TOTP, backup codes)
│   │   └── session.go             # Session locale (token cache)
│   ├── module/                    # Logique module
│   │   ├── creator.go             # Création de module
│   │   ├── manifest.go            # Manipulation manifest.json
│   │   ├── packer.go              # Compression .smp
│   │   ├── linker.go              # Liaison local ↔ distant
│   │   └── validator.go           # Validation module
│   ├── signing/                   # Signature numérique Ed25519
│   │   ├── signer.go              # Génération clés, signature, vérification
│   │   └── keystore.go            # Stockage clés dans le keychain OS
│   ├── audit/                     # Audit de conformité
│   │   ├── auditor.go             # Orchestrateur d'audit
│   │   ├── architecture.go        # Vérification Clean Architecture
│   │   ├── manifest.go            # Validation manifest.json
│   │   └── dependencies.go        # Vérification dépendances
│   ├── debug/                     # Debug de module
│   │   └── debugger.go            # Lancement debug
│   ├── store/                     # Store (fichiers .smp)
│   │   ├── builder.go             # Construction archive
│   │   └── publisher.go           # Publication via API
│   ├── tui/                       # Composants Bubbletea
│   │   ├── app.go                 # Application TUI racine
│   │   ├── styles.go              # Styles Lipgloss
│   │   ├── prompts.go             # Prompts interactifs
│   │   ├── spinner.go             # Indicateur de progression
│   │   └── table.go               # Tableau de sélection
│   └── pkg/                       # Utilitaires
│       ├── fs.go                  # Opérations fichiers
│       ├── git.go                 # Opérations git
│       ├── http.go                # Client HTTP
│       ├── uuid.go                # Génération UUID
│       └── crypto.go              # Chiffrement/hachage
├── go.mod
├── go.sum
├── .goreleaser.yaml               # Configuration GoReleaser
└── README.md
```

### 4.2 Dépendances Go

| Module | Usage | Version |
|--------|-------|---------|
| `github.com/spf13/cobra` | Parsing de commandes | `^1.8.0` |
| `github.com/charmbracelet/bubbletea` | Framework TUI | `^1.2.0` |
| `github.com/charmbracelet/lipgloss` | Styling terminal | `^1.0.0` |
| `github.com/charmbracelet/bubbles` | Composants TUI (spinner, select, input) | `^0.20.0` |
| `github.com/zalando/go-keyring` | Accès keychain système | `^0.2.5` |
| `github.com/google/uuid` | Génération UUID v4 | `^1.6.0` |
| `github.com/BurntSushi/toml` | Parsing TOML | `^1.3.2` |
| `golang.org/x/term` | Détection terminal | `^0.20.0` |
| `net/http` | Client API (stdlib) | — |
| `crypto/aes`, `crypto/cipher` | Chiffrement (stdlib) | — |
| `crypto/ed25519` | Signature numérique Ed25519 (stdlib) | — |
| `archive/zip` | Compression .smp (stdlib) | — |

### 4.3 Pipeline d'exécution

```
Utilisateur
    │
    ▼
Commande (cmd/*.go)
    │ Validation args + flags
    ▼
Service (internal/module/*.go, internal/auth/*.go)
    │ Logique métier
    ▼
Infrastructure (internal/pkg/*.go, internal/config/*.go)
    │ Fichiers, HTTP, keychain, git
    ▼
Sortie TUI (internal/tui/*.go)
    │ Affichage interactif
    ▼
Utilisateur
```

---

## 5. Commandes — Spécification détaillée

---

### 5.1 `sentients init`

#### Purpose

Initialiser un nouveau projet Sentient en clonant le template `protorians/sentient-cms` et en
installant les dépendances.

#### Comportement

1. **Demander le nom du projet** (dossier cible) via input Bubbletea
2. **Détection automatique** des gestionnaires de paquets disponibles sur la machine :
   - `bun` → disponible ?
   - `pnpm` → disponible ?
   - `yarn` → disponible ?
   - `npm` → disponible ?
3. **Proposer le choix** via un sélecteur Bubbletea (liste filtrée aux disponibles)
4. **Cloner** `https://github.com/protorians/sentient-cms` dans `./<nom-projet>/`
5. **Installer les dépendances** avec le gestionnaire sélectionné
6. **Afficher le résumé** : projet initialisé, gestionnaire utilisé, prochaines étapes

#### Contraintes

- Si aucun gestionnaire n'est détecté → erreur explicite avec instructions d'installation
- Si le dossier existe déjà → demander confirmation (écraser / annuler)
- Le clone doit être un shallow clone (`--depth 1`) pour rapidité

#### Sortie TUI

```
? Nom du projet : mon-projet
? Gestionnaire de paquets : bun (recommandé)
  ⠋ Clonage de sentient-cms...
  ⠋ Installation des dépendances...

  ✓ Projet initialisé avec succès

  Prochaines étapes :
    cd mon-projet
    sentients create module
```

---

### 5.2 `sentients create module`

#### Purpose

Créer un nouveau module dans le dossier `external_modules/` à la racine du projet.

#### Comportement

1. **Vérifier le contexte** : être à la racine d'un projet Sentient (fichier `sentient.config.toml` ou détection du dossier `external_modules/`)
2. **Demander le nom du module** via input Bubbletea
3. **Valider le nom** : kebab-case, pas de caractères spéciaux, longueur 3-64
4. **Créer la structure** :

```
external_modules/<module-name>/
├── manifest.json
├── index.tsx
├── components/
│   └── .gitkeep
├── hooks/
│   └── .gitkeep
├── services/
│   └── .gitkeep
└── README.md
```

5. **Générer `manifest.json`** avec :

```json
{
  "schemaVersion": 1,
  "id": "<module-name>",
  "domain": "mod.sentients.<module-name>",
  "key": "<MODULE_NAME_UPPER>",
  "name": "<Nom du module>",
  "description": "",
  "version": "0.1.0",
  "icon": "PuzzleIcon",
  "type": "EXTERNAL",
  "entry": "index.tsx",
  "uri": "/<module-name>",
  "token": "<UUID v4 généré>",
  "publisher": {
    "id": "",
    "name": ""
  },
  "platforms": {
    "web": { "supported": true, "modes": ["web"] },
    "desktop": { "supported": false },
    "mobile": { "supported": false }
  },
  "managerCompatibility": { "min": "0.0.0", "max": "*.x" },
  "apiCompatibility": { "min": "0.0.0", "max": "*.x" },
  "permissions": [],
  "apiScopes": [],
  "capabilities": {
    "needsNetwork": true,
    "supportsOffline": false,
    "requiresOrganization": false,
    "requiresAuthenticatedUser": true
  },
  "isEnabled": true,
  "isDefault": false,
  "requirements": {},
  "dependencies": {
    "@sentients/sdk": "workspace:*"
  },
  "widgets": [],
  "routines": [],
  "menu": { "items": [] }
}
```

6. **Générer `index.tsx`** avec un squelette conforme aux conventions du manager :

```tsx
import type { ModuleDeclarationInterface } from "@/modules";

const declaration: ModuleDeclarationInterface = {
  name: "<ModuleName>",
  description: "",
  render: async () => {
    const mod = await import("./components");
    return mod.default;
  },
};

export default declaration;
```

7. **Générer `README.md`** avec les métadonnées du module
8. **Afficher le résumé** : module créé, emplacement, prochaines étapes

#### Contraintes

- Le token UUID est **unique** et généré à la création
- Le nom du module ne peut pas entrer en conflit avec un module existant
- Le `key` (ModuleEnum) est généré en UPPER_SNAKE_CASE du nom

#### Sortie TUI

```
? Nom du module : blog-manager
? Description du module : Gestion de blog et d'articles

  ✓ Module créé : external_modules/blog-manager/
  ✓ Token généré : a1b2c3d4-e5f6-7890-abcd-ef1234567890
  ✓ manifest.json initialisé
  ✓ index.tsx initialisé

  Prochaines étapes :
    sentients connect
    sentients pack blog-manager
    sentients publish
```

---

### 5.3 `sentients connect`

#### Purpose

Authentifier le développeur avec son compte `sentient-connect` et stocker les credentials de
manière sécurisée.

#### Comportement

1. **Vérifier si déjà connecté** : credentials existantes dans le keychain
   - Si oui → afficher le statut et demander si reconnexion souhaitée
2. **Demander l'email** via input Bubbletea
3. **Demander le mot de passe** via input Bubbletea (masqué)
4. **Envoyer les credentials** à l'API `sentient-connect` (`POST /auth/sign-in`)
5. **Vérifier la réponse** :
   - **Succès sans MFA** → stocker le token Bearer + refresh token dans le keychain
   - **MFA requis** (`mfaRequired: true`) → enchaîner sur l'étape MFA
6. **MFA** (si requis) :
   - Afficher les facteurs disponibles (TOTP, backup codes)
   - Selon le facteur choisi :
     - **TOTP** : demander le code 6 chiffres → `POST /mfa/totp/verify`
     - **Backup code** : demander le code → `POST /mfa/recovery/verify`
   - Valider → stocker les credentials
7. **Afficher le résumé** : connecté en tant que `email`, rôle, organisation

#### Stockage des credentials

| Donnée | Emplacement | Chiffrement |
|--------|-------------|-------------|
| `access_token` | Keychain (`sentient-cli.access_token`) | Oui (keychain natif) |
| `refresh_token` | Keychain (`sentient-cli.refresh_token`) | Oui (keychain natif) |
| `expires_at` | Keychain (`sentient-cli.expires_at`) | Non (timestamp) |
| `user.email` | Keychain (`sentient-cli.user_email`) | Non |
| `user.id` | Keychain (`sentient-cli.user_id`) | Non |
| `mfa_secret` | Keychain (`sentient-cli.mfa_secret`) | Oui (keychain natif) |

#### Sécurité

- **Plus jamais** de credentials en clair sur disque
- Les tokens sont automatiquement rafraîchis via `refresh_token` si expirés
- Le `mfa_secret` (si TOTP enrollment local) est chiffré dans le keychain
- Après 5 échecs de connexion → temporaire (5 min) avec message clair

#### Sortie TUI

```
? Email : dev@example.com
? Mot de passe : ********
  ⠋ Vérification des identifiants...

? Code MFA (TOTP) : 123456
  ⠋ Vérification du code...

  ✓ Connecté en tant que dev@example.com
    Rôle : Developer
    Organisation : Mon Organisation
    Token expire le : 2026-09-17 14:30:00 UTC
```

---

### 5.4 `sentients disconnect`

#### Purpose

Supprimer toutes les credentials stockées et déconnecter le développeur.

#### Comportement

1. **Vérifier si connecté** : credentials présentes dans le keychain
   - Si non connecté → message informatif, rien à faire
2. **Demander confirmation** ( Bubbletea confirm )
3. **Supprimer** toutes les entrées du keychain :
   - `sentient-cli.access_token`
   - `sentient-cli.refresh_token`
   - `sentient-cli.expires_at`
   - `sentient-cli.user_email`
   - `sentient-cli.user_id`
   - `sentient-cli.mfa_secret`
4. **Invalider le token** côté serveur (`POST /auth/sign-out`)
5. **Afficher confirmation**

#### Sortie TUI

```
? Confirmer la déconnexion : Oui
  ✓ Déconnecté avec succès
    Toutes les credentials ont été supprimées.
```

---

### 5.5 `sentients pack`

#### Purpose

Construire le build d'un module et créer une archive `.smp` compressée.

#### Comportement

1. **Identifier le module** :
   - Si un argument `<module>` est fourni → l'utiliser
   - Sinon → lister les modules dans `external_modules/` via un sélecteur Bubbletea
2. **Vérifier l'existence** du module et de ses fichiers essentiels (`manifest.json`, `index.tsx`)
3. **Valider le `manifest.json`** (champs requis : `id`, `name`, `version`, `token`)
4. **Construire les chemins** :
   - Source module : `external_modules/<module>/`
   - Source assets : `public/assets/<module>/` (si existe)
   - Destination : `.sentients/build/`
5. **Créer l'archive ZIP** :
   - Nom : `<module>-<version>.smp` (le `.smp` est un ZIP renommé)
   - Contenu : dossiers `external_modules/<module>/` et `public/assets/<module>/` (si existe)
   - Préfixe dans l'archive : `external_modules/<module>/` et `public/assets/<module>/`
6. **Déplacer** l'archive vers `.sentients/build/`
7. **Afficher le résumé** : taille de l'archive, emplacement

#### Structure de l'archive `.smp`

```
<smp-file>.smp (ZIP)
├── external_modules/<module>/
│   ├── manifest.json
│   ├── index.tsx
│   ├── components/
│   ├── hooks/
│   └── ...
└── public/assets/<module>/    (optionnel)
    └── ...
```

#### Contraintes

- Le dossier `.sentients/build/` est créé automatiquement s'il n'existe pas
- Si une archive du même nom existe → demander confirmation (écraser)
- Le `manifest.json` doit être valide avant le pack
- La taille maximale de l'archive est de 50 MB (limite store)

#### Sortie TUI

```
? Sélectionner le module : blog-manager
  ⠋ Validation du manifest.json...
  ⠋ Construction de l'archive...
  ⠋ Déplacement vers .sentients/build/

  ✓ Archive créée avec succès
    Module : blog-manager v0.1.0
    Fichier : .sentients/build/blog-manager-0.1.0.smp
    Taille : 12.4 KB
```

---

### 5.6 `sentients publish`

#### Purpose

Construire et publier un module dans le store via l'API `sentient-connect`.

#### Comportement

1. **Vérifier l'authentification** : token Bearer valide dans le keychain
   - Si non connecté → `sentients connect` automatique
2. **Identifier le module** : sélecteur Bubbletea si non fourni
3. **Vérifier le `manifest.json`** :
   - Si les métadonnées sont incomplètes (champs vides) → **demander** :
     - `name` : nom affiché du module
     - `description` : description courte
     - `publisher.id` : identifiant développeur
     - `publisher.name` : nom affiché du développeur
   - Proposer de mettre à jour le `manifest.json` local
4. **Exécuter `sentients pack`** en interne (construction de l'archive)
5. **Envoyer l'archive** à l'API :
   - `POST /store/modules/publish`
   - Headers : `Authorization: Bearer <token>`
   - Body : multipart/form-data avec l'archive `.smp` + métadonnées JSON
6. **Gérer la réponse** :
   - **Succès** → afficher l'URL du module dans le store
   - **Conflit** (version existante) → demander si bump de version souhaité
   - **Erreur** → afficher le message d'erreur détaillé
7. **Mettre à jour le `manifest.json`** local avec la version publiée

#### Contraintes

- L'authentification est **obligatoire**
- La version doit être supérieure à la dernière version publiée (SemVer)
- Le module doit passer l'audit (`sentients audit`) avant la publication
- Si l'audit échoue → proposer de corriger avant de publier

#### Sortie TUI

```
  ✓ Authentifié : dev@example.com
? Sélectionner le module : blog-manager

  Métadonnées du module :
    Nom : Blog Manager
    Description : Gestion de blog et d'articles
    Version : 0.1.0

? Confirmer la publication : Oui
  ⠋ Validation du manifest.json...
  ⠋ Construction de l'archive...
  ⠋ Publication sur le store...
  ✓ Publié avec succès

  Module : blog-manager v0.1.0
  URL : https://store.sentient.dev/modules/blog-manager
```

---

### 5.7 `sentients link`

#### Purpose

Lier un module créé dans `sentient-connect` avec le module en local.

#### Comportement

1. **Vérifier l'authentification** (sinon → `sentients connect`)
2. **Lister les modules locaux** dans `external_modules/` via sélecteur Bubbletea
3. **Lister les modules en ligne** via API :
   - `GET /store/modules` (modules du développeur)
   - Afficher dans un tableau Bubbletea avec : nom, version, statut
4. **Demander le token du module en ligne** via input Bubbletea
5. **Valider le token** via API :
   - `GET /store/modules/<token>`
   - Vérifier que le module existe et appartient au développeur
6. **Mettre à jour le `manifest.json` local** :
   - Ajouter/mettre à jour le champ `token` avec le token distant
   - Ajouter les métadonnées distantes si absentes localement
7. **Afficher le résumé** : module lié (local ↔ distant)

#### Sortie TUI

```
? Sélectionner le module local : blog-manager
? Token du module en ligne : m_abc123def456
  ⠋ Vérification du module distant...

  ✓ Module lié avec succès
    Local : external_modules/blog-manager/
    Distant : m_abc123def456 (Blog Manager v0.1.0)
```

---

### 5.8 `sentients unlink`

#### Purpose

Délier un module local de son correspondant dans `sentient-connect`.

#### Comportement

1. **Vérifier l'authentification** (sinon → `sentients connect`)
2. **Lister les modules locaux** ayant un token distant dans leur `manifest.json`
3. **Afficher la liste** via sélecteur Bubbletea (modules liés uniquement)
4. **Demander confirmation**
5. **Supprimer le champ `token`** du `manifest.json` local
6. **Afficher confirmation**

#### Sortie TUI

```
? Sélectionner le module à délier : blog-manager
  Actuellement lié à : m_abc123def456 (Blog Manager v0.1.0)

? Confirmer la déliaison : Oui
  ✓ Module délié avec succès
    external_modules/blog-manager/ n'est plus lié à un module distant.
```

---

### 5.9 `sentients debug <module>`

#### Purpose

Lancer le debug d'un ou tous les modules dans `external_modules/`.

#### Comportement

1. **Analyser l'argument** :
   - Si `<module>` est fourni → debug uniquement ce module
   - Sinon → debug **tous** les modules dans `external_modules/`
2. **Vérifier l'existence** du ou des modules
3. **Lancer le processus de debug** :
   - Exécuter le build du module avec les flags de debug (`--debug`, source maps)
   - Surveiller les erreurs en temps réel
   - Afficher les logs du module dans un viewer TUI (scrollable)
4. **Mode single module** :
   - Compiler le module en mode dev
   - Afficher les erreurs de compilation
   - Afficher les erreurs d'exécution (si applicable)
5. **Mode all modules** :
   - Compiler tous les modules en parallèle
   - Afficher un tableau de statut (nom, statut, erreurs)
   - Détecter les conflits entre modules

#### Sortie TUI (single)

```
  Debug : blog-manager
  ⠋ Compilation en mode debug...
  ✓ Compilation réussie

  ┌─────────────────────────────────────────────┐
  │ [14:30:01] blog-manager: Module chargé       │
  │ [14:30:01] blog-manager: Routes enregistrées │
  │ [14:30:02] blog-manager: Aucune erreur       │
  └─────────────────────────────────────────────┘
```

#### Sortie TUI (all)

```
  Debug de tous les modules
  ┌──────────────────┬──────────┬─────────────┐
  │ Module           │ Statut   │ Erreurs     │
  ├──────────────────┼──────────┼─────────────┤
  │ blog-manager     │ ✓ OK     │ 0           │
  │ billing          │ ✓ OK     │ 0           │
  │ calendar         │ ⚠ 2      │ 2 warnings  │
  │ crm              │ ✓ OK     │ 0           │
  └──────────────────┴──────────┴─────────────┘
```

---

### 5.10 `sentients audit <module>`

#### Purpose

Auditer la conformité d'un ou tous les modules par rapport aux règles du système Sentient.

#### Comportement

1. **Analyser l'argument** :
   - Si `<module>` est fourni → audit uniquement ce module
   - Sinon → audit **tous** les modules dans `external_modules/`
2. **Vérifier l'existence** du ou des modules
3. **Exécuter les vérifications** (pour chaque module) :

#### Vérifications d'audit

| Catégorie | Règle | Sévérité |
|-----------|-------|----------|
| **Clean Architecture** | Pas de dépendances directes entre couches interdites | ERROR |
| **Clean Architecture** | Les hooks ne contiennent pas de logique métier | WARNING |
| **Clean Architecture** | Les services ne contiennent pas de JSX | ERROR |
| **Clean Architecture** | Les composants n'importent pas directement les services | WARNING |
| **manifest.json** | Champ `id` présent et non vide | ERROR |
| **manifest.json** | Champ `name` présent et non vide | ERROR |
| **manifest.json** | Champ `version` au format SemVer valide | ERROR |
| **manifest.json** | Champ `entry` pointe vers un fichier existant | ERROR |
| **manifest.json** | Champ `domain` au format `mod.sentients.<name>` | WARNING |
| **manifest.json** | Champ `token` présent (UUID valide) | ERROR |
| **manifest.json** | `permissions` est un tableau | WARNING |
| **manifest.json** | `dependencies` listent des packages existants | ERROR |
| **index.tsx** | Fichier existe et exporte un `ModuleDeclarationInterface` | ERROR |
| **index.tsx** | Le champ `name` correspond au manifest | WARNING |
| **index.tsx** | La fonction `render` est asynchrone | ERROR |
| **requirements** | Toutes les requirements listées dans le manifest existent dans `external_modules/` | ERROR |
| **dependencies** | Toutes les dépendances npm sont installées | ERROR |
| **dependencies** | Pas de dépendances en double | WARNING |
| **assets** | Les fichiers dans `public/assets/<module>/` existent | WARNING |

4. **Générer le rapport** :
   - Afficher les résultats dans un tableau TUI
   - Colorer par sévérité (rouge = ERROR, orange = WARNING, vert = OK)
   - Résumé : X erreurs, Y warnings, Z modules audités
5. **Optionnel** : exporter le rapport en JSON (`--output json`)

#### Sortie TUI

```
  Audit : blog-manager
  ┌──────────────────────┬──────────┬──────────────────────────┐
  │ Catégorie            │ Règle    │ Statut                   │
  ├──────────────────────┼──────────┼──────────────────────────┤
  │ manifest.json        │ id       │ ✓ Présent                │
  │ manifest.json        │ name     │ ✓ Présent                │
  │ manifest.json        │ version  │ ✓ SemValide              │
  │ manifest.json        │ token    │ ✓ UUID valide            │
  │ manifest.json        │ entry    │ ✓ Fichier existe         │
  │ index.tsx            │ export   │ ✓ Défaut exporté         │
  │ index.tsx            │ render   │ ✓ Async                  │
  │ requirements         │ exist    │ ⚠ organization non trouvé│
  │ dependencies         │ install  │ ✓ Toutes installées      │
  │ Clean Architecture   │ couche   │ ✓ Conforme               │
  └──────────────────────┴──────────┴──────────────────────────┘

  Résumé : 0 erreurs, 1 warning
```

---

### 5.11 `sentients help`

#### Purpose

Afficher l'aide contextuelle de la CLI.

#### Comportement

1. **Sans argument** : afficher la liste de toutes les commandes avec descriptions
2. **Avec une commande** : afficher l'aide détaillée de cette commande (flags, exemples)

#### Sortie TUI (sans argument)

```
Sentient CLI — Outil de développement pour les modules Sentient

Usage:
  sentients <commande> [options]

Commandes disponibles:
  init                  Initialiser un projet Sentient
  create module         Créer un nouveau module
  connect               Se connecter à Sentient Connect
  disconnect            Se déconnecter
  pack                  Construire l'archive d'un module (.smp)
  sign keygen           Générer une paire de clés de signature
  sign <module>         Signer l'archive d'un module
  sign verify <module>  Vérifier la signature d'un module
  publish               Publier un module sur le store
  link                  Lier un module local à un module distant
  unlink                Délier un module
  debug <module>        Déboguer un module
  audit <module>        Auditer la conformité d'un module
  help                  Afficher cette aide
  -v, --version         Afficher la version

Options globales:
  --verbose             Activer les logs détaillés
  --no-color            Désactiver les couleurs
  --help                Afficher l'aide
  --version             Afficher la version

Exemples:
  sentients init
  sentients create module
  sentients connect
  sentients pack blog-manager
  sentients sign blog-manager
  sentients publish blog-manager
  sentients audit
```

---

### 5.12 `sentients -v` / `sentients --version`

#### Purpose

Afficher la version actuelle de la CLI.

#### Comportement

1. Lire la version compilée dans le binaire (via `ldflags`)
2. Afficher : `sentients v<version> (<os>/<arch>) <commit>`

#### Sortie

```
sentients v0.1.0 (darwin/arm64) abc1234
```

---

### 5.13 `sentients sign`

#### Purpose

Gérer les signatures numériques Ed25519 des modules : générer des clés, signer les archives `.smp`
et vérifier les signatures. La signature garantit l'intégrité et l'authenticité des modules
avant publication.

#### Sous-commandes

| Sous-commande | Description |
|---------------|-------------|
| `sentients sign keygen` | Générer une paire de clés Ed25519 et la stocker dans le keychain |
| `sentients sign <module>` | Signer l'archive `.smp` d'un module |
| `sentients sign verify <module>` | Vérifier la signature d'un module |

---

##### 5.13.1 `sentients sign keygen`

###### Comportement

1. **Vérifier si des clés existent déjà** dans le keychain
   - Si oui → afficher le fingerprint de la clé publique et demander régénération
2. **Générer une paire de clés Ed25519** (`crypto/ed25519`)
3. **Stocker** la clé privée et la clé publique dans le keychain système
   - Clé privée : `sentient-cli-signing.signing_private_key`
   - Clé publique : `sentient-cli-signing.signing_public_key`
   - Fallback : fichier chiffré `~/.sentient-cli/signing.enc` (AES-256-GCM)
4. **Afficher** le fingerprint SHA-256 de la clé publique (hex 64 caractères)

###### Contraintes

- La clé privée n'est **jamais** affichée à l'écran
- Si des clés existent déjà, la régénération écrase les précédentes après confirmation
- En mode non interactif, régénère silencieusement (pour CI/CD)

###### Sortie TUI

```
  ✓ Paire de clés Ed25519 générée avec succès
    Fingerprint : a1b2c3d4e5f6... (SHA-256 de la clé publique)
    Clé privée  : stockée dans le keychain système
    Clé publique : stockée dans le keychain système
```

---

##### 5.13.2 `sentients sign <module>`

###### Comportement

1. **Vérifier le contexte** : être à la racine d'un projet Sentient
2. **Identifier le module** : argument `<module>` ou sélecteur Bubbletea
3. **Charger le `manifest.json`** du module pour obtenir la version
4. **Vérifier que l'archive `.smp` existe** dans `.sentients/build/`
   - Si absente → erreur avec suggestion d'exécuter `sentients pack <module>`
5. **Charger la clé privée** depuis le keychain
   - Si absente → erreur avec suggestion d'exécuter `sentients sign keygen`
6. **Signer l'archive** :
   - Lire le contenu de l'archive `.smp`
   - Signer avec `ed25519.Sign(privateKey, archiveData)`
   - Écrire la signature dans `<archive>.sig` (même dossier que l'archive)
7. **Afficher le résumé** : module, version, fingerprint du signataire, chemin du `.sig`

###### Contraintes

- L'archive `.smp` doit exister (résultat de `sentients pack`)
- La clé privée doit exister dans le keychain
- Si un fichier `.sig` existe déjà pour cette archive → demander confirmation (écraser)
- Le fichier `.sig` est un binaire contenant uniquement la signature Ed25519 (64 octets)

###### Sortie TUI

```
? Sélectionner le module : blog-manager
  ⠋ Chargement de la clé de signature…
  ⠋ Signature de l'archive…

  ✓ Archive signée avec succès
    Module   : blog-manager v0.1.0
    Archive  : .sentients/build/blog-manager-0.1.0.smp
    Signature : .sentients/build/blog-manager-0.1.0.smp.sig
    Signataire : a1b2c3d4... (fingerprint SHA-256)
```

---

##### 5.13.3 `sentients sign verify <module>`

###### Comportement

1. **Vérifier le contexte** : être à la racine d'un projet Sentient
2. **Identifier le module** : argument `<module>` ou sélecteur Bubbletea
3. **Charger le `manifest.json`** du module pour obtenir la version
4. **Vérifier que l'archive `.smp` et le fichier `.sig` existent**
5. **Charger la clé publique** depuis le keychain
   - Si absente → erreur avec suggestion d'exécuter `sentients sign keygen`
6. **Vérifier la signature** :
   - Lire l'archive `.smp` et le fichier `.sig`
   - Vérifier avec `ed25519.Verify(publicKey, archiveData, signature)`
7. **Afficher le résultat** : ✓ Signature valide ou ✗ Signature invalide

###### Contraintes

- L'archive `.smp` ET le fichier `.sig` doivent exister
- La clé publique doit exister dans le keychain
- En cas de signature invalide → afficher un message d'erreur explicite (possiblement archive corrompue ou clé incorrecte)

###### Sortie TUI (valide)

```
? Sélectionner le module : blog-manager
  ⠋ Vérification de la signature…

  ✓ Signature valide
    Module    : blog-manager v0.1.0
    Archive   : .sentients/build/blog-manager-0.1.0.smp
    Signataire : a1b2c3d4...
```

###### Sortie TUI (invalide)

```
? Sélectionner le module : blog-manager
  ⠋ Vérification de la signature…

  ✗ Signature invalide
    Module   : blog-manager v0.1.0
    Archive  : .sentients/build/blog-manager-0.1.0.smp
    → L'archive a pu être modifiée ou la clé de vérification est incorrecte.
```

---

## 6. Modèle de données local

### 6.1 Fichier `.sentient-cli.toml` (optionnel)

Placé à la racine du projet Sentient, ce fichier permet de configurer la CLI.

```toml
[project]
name = "mon-projet"
package_manager = "bun"

[publish]
default_registry = "https://store.sentient.dev"
auto_audit = true

[debug]
verbose = false
log_level = "info"
```

### 6.2 Fichier `manifest.json` (par module)

Le `manifest.json` est le fichier de métadonnées de chaque module. Voir la section
`sentients create module` pour le schéma complet.

### 6.3 Keychain — Hiérarchie des clés

```
sentient-cli/
├── access_token      # Token Bearer JWT
├── refresh_token     # Token de rafraîchissement
├── expires_at        # Timestamp d'expiration
├── user_id           # ID du développeur
├── user_email        # Email du développeur
└── mfa_secret        # Secret TOTP (si enrollment local)

sentient-cli-signing/
├── signing_public_key   # Clé publique Ed25519 (fingerprint du développeur)
└── signing_private_key  # Clé privée Ed25519 (signature des archives .smp)
```

---

## 7. Sécurité

### 7.1 Stockage des credentials

| Mécanisme | Plateforme | Implémentation |
|-----------|------------|----------------|
| macOS Keychain | macOS | `security` CLI ou `go-keyring` |
| Secret Service | Linux | D-Bus + `libsecret` via `go-keyring` |
| Credential Manager | Windows | `cmdkey` ou `Credential Manager` via `go-keyring` |

### 7.2 Chiffrement des archives

- Les archives `.smp` ne contiennent **jamais** de credentials, tokens ou données sensibles
- Le `manifest.json` ne contient que les métadonnées publiques du module
- Le token UUID est un identifiant public, pas un secret

### 7.3 Communication réseau

- Toutes les communications avec `sentient-connect` utilisent **HTTPS** (TLS 1.3)
- Les tokens sont transmis via le header `Authorization: Bearer <token>`
- Pas de credentials dans les query parameters
- Validation SSL stricte (pas de `--insecure`)

### 7.4 Rate limiting

| Action | Limite | Durée |
|--------|--------|-------|
| Tentative de connexion | 5 | 5 min |
| Publication | 10 | 1 heure |
| Appels API | 100 | 1 minute |

### 7.5 MFA

- Si le compte développeur a la MFA activée, `sentients connect` **exige** la vérification
- Le secret TOTP peut être géré côté serveur (recommandé) ou stocké localement (optionnel)
- Les backup codes sont utilisables uniquement en secours

### 7.6 Signature numérique

- Les clés de signature sont stockées dans le keychain OS (service `sentient-cli-signing`)
- La clé privée n'est **jamais** affichée à l'écran ni exportée
- Fallback : fichier chiffré `~/.sentient-cli/signing.enc` (AES-256-GCM) quand le keychain n'est pas disponible
- L'algorithme utilisé est **Ed25519** (signatures compactes de 64 octets, clés de 32 octets)
- Les fichiers `.sig` sont des binaires contenant uniquement la signature Ed25519
- La vérification de signature utilise la clé publique stockée dans le keychain
- En cas de perte de clés, `sentients sign keygen` permet de régénérer une nouvelle paire

---

## 8. Communication API

### 8.1 Endpoints `sentient-connect`

| Méthode | Chemin | Description |
|---------|--------|-------------|
| POST | `/auth/sign-in` | Authentification (email + password) |
| POST | `/auth/sign-out` | Déconnexion (invalidation token) |
| POST | `/auth/refresh` | Rafraîchissement du token |
| POST | `/mfa/challenge` | Déclenchement du défi MFA |
| POST | `/mfa/totp/verify` | Vérification code TOTP |
| POST | `/mfa/recovery/verify` | Vérification backup code |
| GET | `/store/modules` | Liste des modules du développeur |
| GET | `/store/modules/:token` | Détail d'un module |
| POST | `/store/modules/publish` | Publication d'un module |
| PUT | `/store/modules/:token` | Mise à jour des métadonnées |

### 8.2 DTOs

#### POST `/auth/sign-in` — `SignInRequest`

| Champ | Type | Requis | Description |
|-------|------|--------|-------------|
| `email` | string | oui | Email du développeur |
| `password` | string | oui | Mot de passe |

#### Réponse `SignInResponse` (succès)

| Champ | Type | Description |
|-------|------|-------------|
| `access_token` | string | JWT Bearer |
| `refresh_token` | string | Token de rafraîchissement |
| `expires_in` | number | Durée de vie en secondes |
| `user` | object | `{ id, email, name, role }` |
| `mfaRequired` | boolean | Si MFA requis |

#### POST `/store/modules/publish` — `PublishRequest`

| Champ | Type | Requis | Description |
|-------|------|--------|-------------|
| `archive` | binary | oui | Fichier `.smp` (multipart) |
| `manifest` | JSON | oui | Contenu du `manifest.json` |

---

## 9. TUI — Composants Bubbletea

### 9.1 Composants réutilisables

| Composant | Usage | Bibliothèque |
|-----------|-------|--------------|
| `Spinner` | Indicateur de progression | `bubbles/spinner` |
| `Select` | Sélection dans une liste | `bubbles/list` |
| `Input` | Saisie de texte | `bubbles/textinput` |
| `Confirm` | Confirmation oui/non | `bubbles/confirm` |
| `Table` | Affichage de données tabulaires | `bubbles/table` |
| `Viewport` | Scroll de contenu | `bubbles/viewport` |
| `Toast` | Notifications | Custom (lipgloss) |

### 9.2 Styles

| Élément | Couleur (dark) | Couleur (light) |
|---------|---------------|-----------------|
| Succès | `#00D26A` | `#00A854` |
| Erreur | `#FF4757` | `#E63946` |
| Warning | `#FFA502` | `#E67E22` |
| Info | `#1E90FF` | `#2980B9` |
| Muted | `#636E72` | `#95A5A6` |
| Accent | `#A855F7` | `#7C3AED` |

### 9.3 Thème

La CLI détecte automatiquement le thème du terminal (dark/light) via `lipgloss.HasDarkBackground()`
et adapte les couleurs en conséquence.

---

## 10. Build & Distribution

### 10.1 GoReleaser

```yaml
# .goreleaser.yaml
version: 2
builds:
  - main: .
    binary: sentients
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - format: tar.gz
    name_template: >-
      sentients-cli_{{ .Version }}_{{ .Os }}_{{ .Arch }}
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: checksums.txt

snapshot:
  name_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

### 10.2 Installation

```bash
# npm / npx (npmjs)
npm install -g @sentients/cli
# ou
npx @sentients/cli

# macOS / Linux
curl -sSL https://get.sentient.dev/cli | sh

# Windows (PowerShell)
iwr -useb https://get.sentient.dev/cli.ps1 | iex

# Go install
go install github.com/protorians/sentient-cms/cli/sentient-cli@latest

# Homebrew (à créer)
brew install protorians/sentient/sentient-cli
```

### 10.3 Variables de compilation

| Variable | Description | Défaut |
|----------|-------------|--------|
| `main.version` | Version du binaire | `dev` |
| `main.commit` | Hash du commit git | `none` |
| `main.date` | Date de compilation | compilation time |

---

## 11. Gestion des erreurs

### 11.1 Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès |
| `1` | Erreur générique |
| `2` | Erreur d'authentification |
| `3` | Module non trouvé |
| `4` | Manifest invalide |
| `5` | Erreur réseau |
| `6` | Erreur de permission |
| `7` | MFA requis / échoué |
| `10` | Erreur de build |
| `11` | Erreur de publication |
| `12` | Erreur de signature numérique |

### 11.2 Messages d'erreur

Tous les messages d'erreur sont en français et suivent le format :

```
✗ <Catégorie> : <Message détaillé>
  → <Action corrective suggérée>
```

Exemple :

```
✗ Authentification : Token expiré
  → Exécutez 'sentients connect' pour vous reconnecter.
```

---

## 12. Tests

### 12.1 Stratégie de test

| Type | Outil | Couverture cible |
|------|-------|------------------|
| Unit | `testing` stdlib + `testify` | 80% |
| Integration | `testify/suite` | Scénarios complets |
| E2E | `testscript` (txtar) | Toutes les commandes |
| TUI | `bubbletea/teatest` | Composants interactifs |

### 12.2 Scénarios de test critiques

| ID | Scénario |
|----|----------|
| TC-001 | `sentients init` avec bun détecté |
| TC-002 | `sentients init` avec aucun gestionnaire détecté |
| TC-003 | `sentients create module` avec nom invalide |
| TC-004 | `sentients create module` avec nom valide |
| TC-005 | `sentients connect` succès sans MFA |
| TC-006 | `sentients connect` avec MFA TOTP |
| TC-007 | `sentients connect` échec (mauvais identifiants) |
| TC-008 | `sentients disconnect` avec confirmation |
| TC-009 | `sentients pack` module existant |
| TC-010 | `sentients pack` module avec assets |
| TC-011 | `sentients publish` succès |
| TC-012 | `sentients publish` version existante |
| TC-013 | `sentients link` succès |
| TC-014 | `sentients unlink` succès |
| TC-015 | `sentients debug` module unique |
| TC-016 | `sentients debug` tous les modules |
| TC-017 | `sentients audit` module conforme |
| TC-018 | `sentients audit` module avec erreurs |
| TC-019 | `sentients help` sans argument |
| TC-020 | `sentients help` avec commande |
| TC-021 | `sentients -v` affiche la version |
| TC-022 | `sentients sign keygen` génère et stocke les clés Ed25519 |
| TC-023 | `sentients sign <module>` signe l'archive `.smp` et produit un `.sig` |
| TC-024 | `sentients sign verify <module>` vérifie une signature valide |
| TC-025 | `sentients sign verify <module>` échoue sur archive modifiée ou signature invalide |

---

## 13. Roadmap — Découpage Produit

```
Product: sentient-cli v1.0.0
│
├── Release 0.1.0 (MVP)
│   │
│   ├── Epic E-001 : Initialisation & Création
│   │   ├── Story S-001 : `sentients init` (clone + deps)
│   │   └── Story S-002 : `sentients create module`
│   │
│   ├── Epic E-002 : Authentification
│   │   ├── Story S-003 : `sentients connect` (email/password)
│   │   ├── Story S-004 : `sentients connect` (MFA TOTP)
│   │   └── Story S-005 : `sentients disconnect`
│   │
│   └── Epic E-003 : Build & Informations
│       ├── Story S-006 : `sentients pack`
│       └── Story S-007 : `sentients -v` + `sentients help`
│
├── Release 0.2.0 (Store)
│   │
│   ├── Epic E-004 : Publication
│   │   ├── Story S-008 : `sentients publish`
│   │   ├── Story S-009 : `sentients link`
│   │   └── Story S-010 : `sentients unlink`
│   │
│   ├── Epic E-005 : Validation
│   │   ├── Story S-011 : `sentients audit`
│   │   └── Story S-012 : `sentients debug`
│   │
│   └── Epic E-008 : Signature numérique
│       ├── Story S-019 : `sentients sign keygen` (génération clés Ed25519)
│       ├── Story S-020 : `sentients sign <module>` (signature archive .smp)
│       └── Story S-021 : `sentients sign verify <module>` (vérification signature)
│
└── Release 0.3.0 (Qualité)
    │
    ├── Epic E-006 : Expérience développeur
    │   ├── Story S-013 : Mode verbose / logs
    │   ├── Story S-014 : Configuration `.sentient-cli.toml`
    │   └── Story S-015 : Auto-update detection
    │
    └── Epic E-007 : Tests & CI
        ├── Story S-016 : Suite de tests unitaires
        ├── Story S-017 : Tests E2E (testscript)
        └── Story S-018 : Pipeline CI/CD (GoReleaser)
```

---

## 14. Risques

| ID | Risque | Probabilité | Impact | Mitigation |
|----|--------|-------------|--------|------------|
| R-001 | API `sentient-connect` non disponible | Moyenne | Élevé | Mode offline pour les commandes locales (init, create, pack, audit, debug) |
| R-002 | Incompatibilité keychain sur certaines distributions Linux | Moyenne | Moyen | Fallback fichier chiffré (AES-256-GCM) avec warning |
| R-003 | Taille du binaire trop élevée | Faible | Faible | `ldflags -s -w`, UPX compression optionnelle |
| R-004 | Breaking changes API `sentient-connect` | Faible | Élevé | Versioning API, détection automatique de la version |
| R-005 | Conflits de noms de modules | Moyenne | Moyen | Validation stricte, vérification d'unicité avant création |
| R-006 | Archive `.smp` corrompue ou falsifiée | Faible | Élevé | Signature numérique Ed25519 (`sentients sign`), vérification avant publication |
| R-007 | MFA bloquant (appareil perdu) | Faible | Élevé | Backup codes, procédure de récupération via `sentient-connect` web |

---

## 15. ADR (Architecture Decision Records)

### ADR-001 : Go + Bubbletea comme stack technique

**Contexte** : Choisir la stack technique pour la CLI Sentient.

**Options considérées** :
- **Option A** : Go + Cobra + Bubbletea
- **Option B** : Node.js + Commander + Ink
- **Option C** : Rust + Clap + Ratatui
- **Option D** : Python + Click + Rich

**Décision** : Option A — Go + Cobra + Bubbletea

**Justification** :
- Go produit des binaires statiques sans dépendance runtime
- Cobra est le standard Go pour les CLI (utilisé par Hugo, Kubernetes, Docker)
- Bubbletea offre une TUI riche et réactive (framework mature, bonne documentation)
- Écosystème Charm (Lipgloss, Bubbles) très complet
- Compilation croisée simple (Linux, macOS, Windows)
- Performance : démarrage < 100ms

**Conséquences** :
- Pas de JS/TS partagé avec le frontend (mais la CLI est un outil indépendant)
- Apprentissage de Bubbletea pour l'équipe
- Binaire plus gros que Python/Node mais sans dépendance

### ADR-002 : Keychain natif pour les credentials

**Contexte** : Stocker les tokens d'authentification de manière sécurisée.

**Options considérées** :
- **Option A** : Keychain système (go-keyring)
- **Option B** : Fichier chiffré (AES-256-GCM)
- **Option C** : Variable d'environnement

**Décision** : Option A — Keychain natif

**Justification** :
- Sécurité maximale : le OS gère le chiffrement et l'accès
- Pas de mot de passe maître à gérer
- Fallback fichier chiffré pour les environnements sans keychain

**Conséquences** :
- Dépendance au keychain système (fallback nécessaire)
- Sur Linux, nécessite un agent secret (gnome-keyring, KWallet)
