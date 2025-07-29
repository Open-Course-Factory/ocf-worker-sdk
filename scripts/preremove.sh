#!/bin/bash
# Pre-removal script for OCF Worker CLI

set -e

echo "📦 Désinstallation d'OCF Worker CLI..."

# Check if any OCF Worker CLI processes are running
if pgrep -f "ocf-worker-cli" >/dev/null; then
    echo "⚠️  Des processus OCF Worker CLI sont en cours d'exécution"
    echo "   Arrêtez-les avant de continuer ou ils seront terminés"
    
    # Give user a chance to stop processes themselves
    sleep 2
    
    # Kill any remaining processes
    pkill -f "ocf-worker-cli" || true
fi

# Clean up any temporary files (if we create any in the future)
if [ -d "/tmp/ocf-worker-cli" ]; then
    echo "🧹 Nettoyage des fichiers temporaires..."
    rm -rf "/tmp/ocf-worker-cli"
fi

# Note: Package manager will handle removing:
# - /usr/bin/ocf-worker-cli
# - /usr/share/bash-completion/completions/ocf-worker-cli
# - /usr/share/zsh/vendor-completions/_ocf-worker-cli

echo "✅ Préparation de la désinstallation terminée"