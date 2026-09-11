# Rapport d'implémentation — Sentient CLI

> Document de suivi pour implémenter les features au fil des itérations.
> Dernière mise à jour : 2026-09-11 — version courante du code : `dev` (HEAD `cba23bd`, dernière release `v0.0.7`).
> Spécification de référence : `docs/specs/sentient.md` (statut *PLANNING*, pourtant largement implémentée).

---

## 1. Vue d'ensemble

La CLI est un binaire Go (module `github.com/protorians/sentient-cli`, **Go 1.26.0**) qui couvre le
cycle de vie :

```
init → create → develop → debug → audit → pack → sign → link → publish
```

Architecture respectée (TECH-006) : `cmd/` (Cobra, présentation) → `internal/*` (services)
→ `internal/pkg` + `internal/config` (infrastructure), rendu TUI via `internal/tui`.

| Élément | État |
|---------|------|
| 13 commandes Cobra (11 de la spec + `sign` à 3 sous-commandes + helper) | ✅ implémentées |
| 10 packages internes (`auth`, `config`, `module`, `signing`, `audit`, `debug`, `store`, `tui`, `pkg`) | ✅ présents |
| Tests unitaires (`go test ./...`) | ✅ verts (9 packages ok) |
| CI/CD GoReleaser + package npm (`@sentients/cli`) | ✅ en place (releases v0.0.1 → v0.0.7) |
| Messages d'erreur français + codes de sortie spec (§11.1) | ✅ respectés |

**Bilan de couverture spec :** les FR-001 → FR-024, NFR-005/006, SEC-001/002/004/005/006/007/008/009
ont une implémentation (parfois partielle). Le reste des FR (001→024) est couvert côté CLI.

---

## 2. Ce qui est implémenté (par commande)

### `sentient init` (FR-001, FR-002, FR-003)
- Clone shallow de `protorians/sentient-cms` (dossier cible demandé, confirmation/écrasement si existe).
- Détection des package managers `bun → pnpm → yarn → npm` (FR-001) + choix interactif.
- Installation des dépendances (non bloquante, simple `warn` en cas d'échec).
- Écrit `.sentient-cli.toml` (config projet).

### `sentient create module [nom]` (FR-004, FR-005)
- Génère la structure `external_modules/<nom>/` : `manifest.json`, `index.tsx`, `README.md`,
  `components/`, `hooks/`, `services/` (avec `.gitkeep`).
- Token UUID v4 dans le manifest (FR-005), key en `UPPER_SNAKE_CASE`.

### `sentient connect` (FR-006, FR-007, FR-008)
- Sign-in email/mot de passe via `POST /auth/sign-in`, refresh token + expiration.
- MFA TOTP / backup codes via `/mfa/challenge`, `/mfa/totp/verify`, `/mfa/recovery/verify`.
- Credentials stockées dans le keychain OS (`go-keyring`), avec store chiffré de repli.
- Base URL via env `SENTIENT_CONNECT_API` ou `app.config.json`.

### `sentient disconnect` (FR-008, FR-009)
- Invalidation serveur best-effort (`POST /auth/sign-out`) + suppression locale, avec confirmation.

### `sentient pack [module]` (FR-010, FR-011)
- Zip `external_modules/<module>/` + `public/assets/<module>/` → `.sentients/build/<module>-<version>.smp`.
- Validation préalable du manifest (via `Validator`), limite 50 Mo (`MaxArchiveSize`).

### `sentient sign` (FR-021 → FR-024) — `sign keygen` / `sign <module>` / `sign verify <module>`
- Paires de clés **Ed25519**, stockées dans le keychain (service `sentient-cli-signing`).
- Signature binaire 64 octets dans `<archive>.smp.sig` ; vérification sur archive + clé publique.
- Fingerprint SHA-256 de la clé publique (commande `sign` sans argument).

### `sentient publish [module]` (FR-012, FR-013)
- Authentification obligatoire, auto-audit pré-publication (config `auto_audit`), complétion
  interactive des métadonnées (`name`, `description`, `publisher.*`), pack puis upload multipart
  `POST /store/modules/publish`.

### `sentient link` / `sentient unlink` (FR-014, FR-015)
- `link` : liste les modules distants (`GET /store/modules`), valide le token (`GET /store/modules/:token`),
  écrit le token distant dans le `manifest.json` local.
- `unlink` : régénère un token UUID local (déliaison locale ; pas d'appel API de mise à jour).

### `sentient debug [module]` (FR-016)
- Validation + tentative de build de diagnostic (single ou table tous modules), logs formatés.

### `sentient audit [module]` (FR-017, FR-018)
- Audit : Clean Architecture (imports croisés, JSX dans services, index async+render), manifest
  (id/name/version semver/token UUID/entry/domain), index.tsx, requirements, assets.
- Sortie tableau TUI ou JSON (`--output json`), résumé erreurs/warnings.

### `sentient help`, `sentient -v` / `--version` (FR-019, FR-020)
- Aide contextuelle Cobra ; version injectée via ldflags (`main.version/commit/date`).
- Auto-update non bloquant (NFR-006) via GitHub releases (cache 24 h, **notification seule**).

---

## 3. Packages internes

| Package | Rôle | Exports clés |
|---------|------|--------------|
| `auth` | Authentification | `Store`, `Session`, `Connector`, `Authenticator`, `MFAFactor` |
| `config` | Config projet `.sentient-cli.toml` + chemins | `Config`, `Default`, `Load/Save`, `FindProjectRoot`, `ManifestPath` |
| `module` | Logique module | `Manifest`, `Creator`, `Packer`, `Linker`, `Validator` |
| `signing` | Signature Ed25519 | `KeyStore`, `GenerateKeyPair`, `SignArchive`, `VerifySignature`, `Fingerprint`, `FindArchive` |
| `audit` | Audit conformité | `Auditor`, `AuditResult` |
| `debug` | Build de diagnostic | `Debugger`, `DebugResult`, `FormatDebugLogs` |
| `store` | Client store API | `Client` (`ListModules`, `GetModule`, `Publish`) |
| `pkg` | Utilitaires | erreurs+exit codes, crypto AES-256-GCM, fs, git, http, uuid, update |
| `tui` | UI Charm | `AskText/AskSecret/Select/Confirm`, `RunWithSpinner`, `Table`, `NewStyles` |

---

## 4. Écarts, limitations et code incomplet (à corriger en priorité)

### 4.1 Fonctionnalités prévues mais non câblées
- **Fallback keychain → fichier chiffré jamais déclenché.**
  - `auth.NewStore()` (credentials.go:63) et `signing.NewKeyStore()` (keystore.go:43) retournent
    toujours le store keychain. Le fallback AES-256-GCM (`credentials.enc` / `signing.enc`) existe
    mais n'est **jamais activé** si le keychain n'est pas disponible (ce que promettent la spec §7.6,
    R-002 et l'aide `sentient sign`).
  - `signing.NewKeyStoreVolatile()` et `auth.NewStoreVolatile()` sont du code mort (non appelé).
- **Passphrases dures codées** pour le chiffrement de repli : `"sentient-cli-fallback-v1"`
  (credentials.go:77) et `"sentient-cli-signing-v1"` (keystore.go:55) → AES = SHA-256 de la passphrase
  (pkg/crypto.go), sans KDF. Brute-forçable pour quiconque lit le binaire. À remplacer par une
  passphrase dérivée/stockée (TODO sécurité).
- **`sign verify` / MFA** : les erreurs du challenge MFA sont silencieusement ignorées
  (mfa.go:30-39) — un endpoint `/mfa/challenge` en panne est masqué.

### 4.2 Régressions de couverture spec (FR)
- **FR-012 « conflit de version » non géré par `publish`** : le cas « version existante → bump SemVer »
  (§5.6 étape 6) n'est pas implémenté (pas de gestion du code conflict côté prix).
- **FR-014 `link`** : la spec exige de renseigner aussi métadonnées distantes absentes (ébauche §5.7) ;
  seul le token est réécrit.
- **FR-017/018 `audit`** : plusieurs règles spec absentes du code :
  - `permissions` doit être un tableau (WARNING) — non vérifié ;
  - `dependencies` installées (ERROR)/en double (WARNING) — le contrôle doublon est **mort** :
    itération sur un `map[string]string` (auditor.go:190-201) qui ne peut jamais exposer 2 fois la clé ;
  - assets : la doc (auditor.go:204) promet une vérif des assets référencés, le code ne teste que
    « dossier non vide ».
- **FR-016 `debug`** : la découverte du script de build lit le `package.json` **racine** du projet
  (debugger.go:128-144) et échoue si module sans script → marqué `OK` sans aucun build réel. Heuristique
  par `strings.Contains` fragile.

### 4.3 Heuristiques fragiles (fiabilité)
- **`Validator.containsDefaultExport`** (validator.go:138) : regex `export\s+default\s+declaration`
  ne matche QUE le texte littéral généré par `creator.go`. `export default function Foo` / `export default () =>`
  échouent. → faux négatifs/positifs.
- **SemVer checké par regex** (validator.go:132-136), pas de bibliothèque semver (version `*.x` gérée
  à la main).
- **Audit Clean Architecture par `strings.Contains`** sur les chemins d'import + heuristique
  « `<`+`>`+`jsx/tsx` » (auditor.go:148) — pas de vrais parseurs TS/JSX.
- **`Linker.LinkedModules`** : distinction local/distant par **format de chaîne** (UUID ⇒ local,
  non-UUID ⇒ distant) (linker.go:94-95) — fragile si les tokens distants changent de format.

### 4.4 Divers
- `signing.SignResult` — type exporté **jamais utilisé** (dead code).
- `APIError.Error()` (http.go:93-98) : branches `if/else` identiques (code mort).
- `checkForUpdate` : appel réseau non désactivable en CI (pas d'env pour couper).
- Prompt TUI : couleurs `#A855F7`/`#FFFFFF` dupliquées hors palette (prompts.go:187-189, styles.go:89).
- `creator.go:77` : la `description` est ignorée dans le `index.tsx` généré (toujours `""`).
- `manifest.go` : `widgets`/`routines`/`menu` restent des `json.RawMessage` opaques (schéma non modélisé).
- `disconnect` : pas d'appel API de mise à jour `PUT /store/modules/:token` (la spec en liste un §8.1).

---

## 5. Roadmap — alignement spec (§13) et prochaines itérations suggérées

La spec découpe 3 releases. État actuel : quasi tout le « MVP » et le « Store » sont implémentés.

### Release 0.1.0 (MVP) — ✅ largement faite
`init` ✅ · `create module` ✅ · `connect` (email/password) ✅ · `connect` (MFA) ✅ · `disconnect` ✅ ·
`pack` ✅ · `-v`/`help` ✅

### Release 0.2.0 (Store) — ✅ largement faite
`publish` ✅ · `link` ✅/partiel · `unlink` ✅ · `audit` ✅/partiel · `debug` ✅/partiel ·
`sign keygen/sign/verify` ✅

### Release 0.3.0 (Qualité) — ⏳ à faire
- S-013 mode verbose/logs ✅ déjà présent (`--verbose`, `SENTIENT_CLI_DEBUG`).
- S-014 config `.sentient-cli.toml` ✅ déjà présente.
- S-015 auto-update ✅ partiel (notification seule, pas de download).
- S-016/017/018 tests unitaires + E2E (testscript) + CI — unitaires ✅, **E2E absents**, CI ✅.

### Prochaines itérations proposées (par priorité)
1. **Sécurité/robustesse** : activer le fallback keychain↔fichier chiffré avec détection réelle ;
   remplacer les passphrases dures par une dérivation (KDF) ; désactivable update en CI.
2. **Corriger le code mort / incomplet** : doublons-deps audit, contrôle `permissions` array,
   `SignResult`, branches identiques `APIError.Error()`, description dans `index.tsx`.
3. **Fiabiliser l'audit & validator** : regex default-export élargie, parseur minimal pour les
   heuristiques d'architecture, semver via bibliothèque.
4. **`publish` env.** : gestion du conflit de version (bump SemVer), `PUT /store/modules/:token`.
5. **`debug` env.** : build réel des modules, découverte du script dans le `package.json` du module
   (pas de la racine).
6. **Telese spec** : `sentient test <module>`, `sentient watch` (hot-reload), `sentient deploy`,
   `sentient auth` (OAuth2 PKCE), `sentient marketplace` (§2.4 future scope).

---

## 6. Commandes utiles

```bash
go build -o sentient .
./sentient --help
go test ./...
go vet ./...
goreleaser release --clean   # release multi-plateforme
```

Couverture de test : 9 packages ok (`internal/audit`, `auth`, `config`, `debug`, `module`, `pkg`,
`signing`, `store`, `tui`). `cmd/` n'a pas de tests dédiés (supprimés en 8b243d1).