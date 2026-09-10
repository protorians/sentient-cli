# Changelog

All notable changes to this project will be documented in this file.

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
