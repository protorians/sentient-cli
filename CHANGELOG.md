# Changelog

All notable changes to this project will be documented in this file.


## [v0.0.9] - 2026-09-12

### Changed
- **Alignement API sur les contrats documentés** (`sentient-workspace`) :
  - Enveloppe Raiton `{ message, data, statusCode }` décodée dans `internal/pkg/http.go` (rétro-compat `{code,message}`) ;
  - Auth **à jeton unique** : `POST /api/auth/sign-in` → `{user, token, device}`, `/api/auth/logout`,
    `/api/auth/sessions/refresh` ; le `refresh_token` est retiré du modèle de session ;
  - MFA **gardée** : `POST /api/mfa/challenge` puis `/api/mfa/totp/verify` ou `/api/mfa/recovery/verify`,
    exécutés avec la session après sign-in ;
  - Store sur l'API **developer-store** : `ListModules`/`GetModule`/`UpdateModule`/`Publish` passent par
    `/api/developer-store/modules/**` — la publication suit le pipeline produit → version → artefact
    (checksum SHA-256 hex + signature Ed25519 base64 du `.smp.sig` + poids).

### Docs
- `docs/specs/sentient.md` §5.3/5.4/5.6/5.7, §6.3 et §8 (endpoints + DTOs) réalignés sur les contrats documentés ;
- `docs/rapport-implementation.md` (# connect/disconnect/publish/link) mis à jour.


## [v0.0.8] - 2026-09-12

### Changed
- **Commande renommée `sentient` → `sentients`** : le binaire et toutes les invocations de la CLI utilisent désormais `sentients` (Cobra, GoReleaser, binaire npm, CI).

### Docs
- Mise à jour de la documentation et de la spécification (`README.md`, `docs/specs/sentient.md`, `docs/rapport-implementation.md`) avec la commande `sentients`.


## [v0.0.7] - 2026-09-11

### Features


### Bug Fixes


### Other Changes
- chore(release): add `--no-push` option to `release.sh` for manual push control (70828b3)
- chore(release): remove `[skip ci]` from commit message in `release.sh` (56ac90f)


## [v0.0.6] - 2026-09-11

### Features


### Bug Fixes


### Other Changes
- refactor(ci): replace `version-bump.yml` with `release.sh` (f6171db)

## [v0.0.5] - 2026-09-11

### Features
- feat: update `.goreleaser.yaml` to include binary releases alongside archives (7e9c786)

### Bug Fixes


### Other Changes

## [v0.0.4] - 2026-09-11

### Features
- feat: add comprehensive unit tests and enhance code consistency (39123df)

### Bug Fixes
- fix: add error handling for `os.MkdirAll` in `cmd_test.go` (95f0294)
- fix: add error handling for filesystem operations in tests (b223a43)
- fix: add missing error handling in file and directory operations (4043edd)
- fix: add error handling to tests across multiple packages (9e3fe5d)

### Other Changes
- refactor: improve changelog entry generation in workflows (ef3dbf3)

## [v0.0.3] - 2026-09-10

### Features


### Bug Fixes
- fix: CI Workflow (2c21b14)

### Other Changes


## [v0.0.2] - 2026-09-10

### Features


### Bug Fixes


### Other Changes
- chore: cleanup release workflow and adjust version logic (8996cde)

## [v0.0.1] - 2026-09-10

### Features
- No features


### Bug Fixes
- No bug fixes


### Other Changes
- Maintenance updates


The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.1.0] - 2026-09-10

### Added

- **init** — Cloner `protorians/sentient-cms` + installer les dépendances (détection bun/pnpm/yarn/npm)
- **create module** — Créer un module standardisé dans `external_modules/`
- **connect** — Authentification via sentient-connect (email + mot de passe, MFA TOTP / backup codes)
- **disconnect** — Invalider le token côté serveur et supprimer les credentials
- **pack** — Construire l'archive `.smp` dans `.sentients/build/`
- **sign keygen** — Générer une paire de clés Ed25519 pour la signature
- **sign [module]** — Signer l'archive `.smp` d'un module (Ed25519)
- **sign verify [module]** — Vérifier la signature d'un module
- **publish** — Auditer, packer et publier un module sur le store
- **link / unlink** — Associer un module local à un module distant du store (token)
- **audit** — Auditer la conformité (Clean Architecture, manifest, dépendances, assets)
- **debug** — Valider le module et lancer un build de diagnostic
- Internal packages: auth, config, module, pkg, tui, signing, audit, debug, store
- NPM wrapper package (`@sentients/cli`)
- GitHub Actions workflows (CI, release, version bump)
- AES-256-GCM encrypted fallback for signing keys (`~/.sentient-cli/signing.enc`)
- OS keychain integration for signing keys via `go-keyring`
