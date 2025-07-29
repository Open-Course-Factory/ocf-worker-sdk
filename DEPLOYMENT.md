# 🚀 Guide de Déploiement - OCF Worker CLI

Ce guide détaille le processus complet de déploiement et de distribution d'OCF Worker CLI.

## 📋 Table des matières

- [🏗️ Configuration initiale](#️-configuration-initiale)
- [🔄 Processus de développement](#-processus-de-développement)  
- [📦 Build et packaging](#-build-et-packaging)
- [🚀 Release et distribution](#-release-et-distribution)
- [🧪 Tests et validation](#-tests-et-validation)
- [🔧 Maintenance](#-maintenance)

## 🏗️ Configuration initiale

### 1. Prérequis système

```bash
# Outils requis
sudo apt install -y git curl make
go version  # Go 1.23+
goreleaser version  # Pour les releases

# Installation de GoReleaser si nécessaire
curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh
```

### 2. Configuration GitLab

```bash
# Cloner le projet
git clone https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk.git
cd ocf-worker-sdk

# Vérifier la configuration
make goreleaser-check
make info
```

### 3. Variables GitLab CI/CD

Configurez ces variables dans **Settings > CI/CD > Variables** :

| Variable | Description | Exemple |
|----------|-------------|---------|
| `CI_JOB_TOKEN` | Token GitLab (automatique) | `glpat-xxxx` |
| `GORELEASER_VERSION` | Version GoReleaser | `v1.24.0` |

## 🔄 Processus de développement

### Workflow quotidien

```bash
# 1. Développement
make build                    # Build rapide
make dev-test                 # Test local
make test                     # Tests complets

# 2. Validation
make validate-all             # Validation complète
make lint                     # Code quality

# 3. Test de packaging
make goreleaser-snapshot      # Build complet local
```

### Structure des branches

```shell
main                 # Branch principale (releases)
├── feature/xxx      # Nouvelles fonctionnalités  
├── fix/xxx          # Corrections de bugs
└── release/v0.x.x   # Préparation de releases
```

## 📦 Build et packaging

### Build local

```bash
# Build simple
make build

# Build avec packaging complet
make goreleaser-snapshot
```

### Artifacts générés

```shell
dist/
├── ocf-worker-cli_Linux_x86_64.tar.gz      # Archive Linux AMD64
├── ocf-worker-cli_Linux_arm64.tar.gz       # Archive Linux ARM64
├── ocf-worker-cli_Darwin_x86_64.tar.gz     # Archive macOS Intel
├── ocf-worker-cli_Darwin_arm64.tar.gz      # Archive macOS Apple Silicon
├── ocf-worker-cli_0.1.0_amd64.deb          # Package Debian AMD64
├── ocf-worker-cli_0.1.0_arm64.deb          # Package Debian ARM64
└── checksums.txt                           # Checksums de vérification
```

### Test d'installation locale

```bash
# Test du package Debian
sudo dpkg -i dist/ocf-worker-cli_*_amd64.deb

# Test de l'autocomplétion
ocf-worker-cli [TAB][TAB]

# Test fonctionnel
ocf-worker-cli --version
ocf-worker-cli health
```

## 🚀 Release et distribution

### Release manuelle

```bash
# 1. Validation préalable
make validate-all

# 2. Build de test
make pre-release

# 3. Création du tag
git tag -a v0.1.0 -m "Release v0.1.0"
git push --tags

# Le pipeline GitLab se charge du reste automatiquement
```

### Release automatique

```bash
# Release avec auto-increment de version
make release-complete
```

### Pipeline GitLab

Le pipeline automatise :

1. **Validation** : Tests, lint, security
2. **Build** : Binaires multi-platform
3. **Packaging** : .deb, .tar.gz
4. **Upload** : GitLab Package Registry
5. **Distribution** : Script d'installation
6. **Tests** : Installation sur différents OS

### URLs de distribution

Une fois déployé :

```bash
# Installation universelle
curl -s https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk/-/raw/main/install.sh | bash

# Installation d'une version spécifique
curl -s https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk/-/raw/main/install.sh | bash -s -- --version 0.1.0
```

## 🧪 Tests et validation

### Tests locaux

```bash
# Tests unitaires
make test
make test-coverage

# Validation du script d'installation
make test-install-script

# Test d'installation complète (dans un container)
docker run --rm -v $(pwd):/app -w /app ubuntu:22.04 bash -c "
  apt update && apt install -y curl tar &&
  chmod +x install.sh &&
  ./install.sh --version main
"
```

### Tests automatisés

Le pipeline teste l'installation sur :

- Ubuntu 22.04
- Debian 12
- (Autres OS configurables)

### Validation de la release

```bash
# Vérification des artifacts
ls -la dist/

# Test des checksums
sha256sum -c dist/checksums.txt

# Test d'installation depuis le web
make install-from-web
```

## 🔧 Maintenance

### Mise à jour des dépendances

```bash
# Mise à jour Go modules
make update-deps

# Vérification de sécurité
make security

# Mise à jour GoReleaser
# Éditer .gitlab-ci.yml pour changer GORELEASER_VERSION
```

### Monitoring des releases

```bash
# Vérifier les releases GitLab
curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
     "https://usine.solution-libre.fr/api/v4/projects/$PROJECT_ID/releases"

# Statistiques de téléchargement
curl -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
     "https://usine.solution-libre.fr/api/v4/projects/$PROJECT_ID/packages"
```

### Debug et troubleshooting

```bash
# Informations système
make info
make debug

# Vérification de la configuration
make goreleaser-check

# Logs des pipelines
# Consultez GitLab CI/CD > Pipelines
```

## 📊 Métriques et monitoring

### KPIs à surveiller

- **Taux de succès** des pipelines
- **Temps de build** (objectif < 10 min)
- **Taille des packages** (évolution)
- **Couverture de tests** (maintenir > 80%)

### Alertes recommandées

- Pipeline failed
- Coverage drop > 5%
- Security vulnerabilities
- Dependency updates

## 🔄 Processus de hotfix

```bash
# 1. Créer une branche de hotfix
git checkout -b hotfix/critical-fix

# 2. Développer et tester
make dev-build
make dev-test

# 3. Release rapide
git tag -a v0.1.1 -m "Hotfix v0.1.1"
git push --tags

# 4. Merge back vers main
git checkout main
git merge hotfix/critical-fix
```

## 📚 Documentation utilisateur

### Installation

```bash
# Installation standard
curl -s https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk/-/raw/main/install.sh | bash

# Options d'installation
curl -s .../install.sh | bash -s -- --help
```

### Usage de base

```bash
# Vérifier l'installation
ocf-worker-cli --version

# Aide
ocf-worker-cli --help

# Génération d'une présentation
ocf-worker-cli generate https://github.com/user/repo
```

### Autocomplétion

```bash
# Bash
source /etc/bash_completion.d/ocf-worker-cli

# Zsh  
autoload -U compinit && compinit
```

## 🚨 Sécurité

### Bonnes pratiques

- ✅ **Signature des releases** (planifié)
- ✅ **Checksums de vérification**
- ✅ **Scan de sécurité automatique**
- ✅ **Validation des dépendances**

### Audit de sécurité

```bash
# Scan des vulnérabilités
make security

# Audit des dépendances
go mod why -m all
```

## 📈 Roadmap

### Phase 3 ✅ (Actuelle)

- [x] Script d'installation universel
- [x] Pipeline automatisé
- [x] Tests multi-platform

### Phase 4 🔄 (Prochaine)

- [ ] Repository APT/DEB officiel
- [ ] Homebrew Tap (macOS)
- [ ] Signature GPG des packages

### Phase 5 📋 (Future)

- [ ] Installation via package managers
- [ ] Métriques d'usage
- [ ] Auto-updates

---

## 🆘 Support

- **Documentation** : [README.md](README.md)
- **Issues** : [GitLab Issues](https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk/-/issues)
- **Pipeline** : [GitLab CI/CD](https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk/-/pipelines)

---

**🎉 Prêt pour la distribution universelle !**
