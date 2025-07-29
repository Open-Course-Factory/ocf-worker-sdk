#!/bin/bash
# Script pour générer les fichiers d'autocomplétion

set -e

# Répertoire de travail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
COMPLETION_DIR="$SCRIPT_DIR/completion"

echo "🔧 Génération des fichiers d'autocomplétion..."

# Créer le répertoire de completion s'il n'existe pas
mkdir -p "$COMPLETION_DIR"

# Construire le binaire temporaire
echo "📦 Construction du binaire temporaire..."
cd "$PROJECT_DIR"
go build -o /tmp/ocf-worker-cli-temp ./cli/main/main.go

# Générer l'autocomplétion Bash
echo "🐚 Génération de l'autocomplétion Bash..."
/tmp/ocf-worker-cli-temp completion bash > "$COMPLETION_DIR/bash_completion"

# Générer l'autocomplétion Zsh
echo "🐚 Génération de l'autocomplétion Zsh..."
/tmp/ocf-worker-cli-temp completion zsh > "$COMPLETION_DIR/zsh_completion"

# Générer l'autocomplétion Fish (pour plus tard)
echo "🐚 Génération de l'autocomplétion Fish..."
/tmp/ocf-worker-cli-temp completion fish > "$COMPLETION_DIR/fish_completion"

# Nettoyer
rm -f /tmp/ocf-worker-cli-temp

echo "✅ Fichiers d'autocomplétion générés dans $COMPLETION_DIR"
ls -la "$COMPLETION_DIR"