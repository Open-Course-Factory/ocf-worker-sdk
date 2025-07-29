# 🚀 OCF Worker CLI - Démarrage Rapide

## ⚡ Installation en une ligne

```bash
curl -s https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk/-/raw/main/install.sh | bash
```

## 🎯 Usage immédiat

```bash
# Vérifier l'installation
ocf-worker-cli --version

# Générer une présentation depuis GitHub
ocf-worker-cli generate https://github.com/ttamoud/presentation

# Vérifier la santé du service
ocf-worker-cli health
```

## 📦 Pour les développeurs

### Configuration de l'environnement

```bash
# Cloner le projet
git clone https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk.git
cd ocf-worker-sdk

# Setup automatique de l'environnement
chmod +x setup-dev.sh
./setup-dev.sh

# Build et test
make build
make dev-test
```

### Workflow de développement

```bash
# 1. Développement
make build                    # Build rapide
make test                     # Tests
make lint                     # Quality check

# 2. Packaging
make goreleaser-snapshot      # Build complet avec packages

# 3. Release
make release-complete         # Release automatique
```

## 🔧 Options d'installation

### Installation personnalisée

```bash
# Version spécifique
curl -s .../install.sh | bash -s -- --version 0.1.0

# Sans autocomplétion
curl -s .../install.sh | bash -s -- --no-completion

# Force la réinstallation
curl -s .../install.sh | bash -s -- --force
```

### Installation depuis les packages

```bash
# Debian/Ubuntu (.deb)
wget https://usine.solution-libre.fr/.../releases/latest/download/ocf-worker-cli_amd64.deb
sudo dpkg -i ocf-worker-cli_amd64.deb

# Extraction manuelle (.tar.gz)
wget https://usine.solution-libre.fr/.../releases/latest/download/ocf-worker-cli_Linux_x86_64.tar.gz
tar -xzf ocf-worker-cli_Linux_x86_64.tar.gz
sudo install ocf-worker-cli /usr/local/bin/
```

## 🎨 Exemples d'usage

### Génération basique

```bash
# Depuis un dépôt complet
ocf-worker-cli generate https://github.com/user/my-slides

# Depuis un sous-dossier
ocf-worker-cli generate https://github.com/user/repo --subfolder presentations/slides
```

### Options avancées

```bash
# Avec sortie personnalisée
ocf-worker-cli generate https://github.com/user/repo \
  --output ./my-presentation \
  --wait-timeout 20m \
  --verbose

# Ouverture automatique
ocf-worker-cli generate https://github.com/user/repo --open
```

### Gestion des jobs

```bash
# Lister les jobs
ocf-worker-cli jobs list

# Status d'un job
ocf-worker-cli jobs status <job-id>

# Logs d'un job
ocf-worker-cli jobs logs <job-id>
```

### Gestion des thèmes

```bash
# Lister les thèmes disponibles
ocf-worker-cli themes list

# Installer un thème
ocf-worker-cli themes install @slidev/theme-seriph

# Auto-installation pour un job
ocf-worker-cli themes auto-install <job-id>
```

## 🔍 Troubleshooting

### Problèmes courants

```bash
# Service inaccessible
ocf-worker-cli health
# → Vérifiez l'URL de l'API avec --api-url

# Timeout de génération
ocf-worker-cli generate <url> --wait-timeout 30m
# → Augmentez le timeout pour les gros dépôts

# Problème d'autocomplétion
source /etc/bash_completion.d/ocf-worker-cli  # Bash
autoload -U compinit && compinit              # Zsh
```

### Debug et logs

```bash
# Mode verbeux
ocf-worker-cli generate <url> --verbose

# Informations système
ocf-worker-cli --version
ocf-worker-cli health --verbose
```

## 📚 Documentation complète

- **API Reference** : `ocf-worker-cli --help`
- **Guide de développement** : [DEPLOYMENT.md](DEPLOYMENT.md)
- **Repository** : [GitLab Repository](https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk)

## 🆘 Support

- **Issues** : [GitLab Issues](https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk/-/issues)
- **Releases** : [GitLab Releases](https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk/-/releases)

---

**🎉 Prêt à créer des présentations Slidev magnifiques !**
