#!/bin/bash
# Post-installation script for OCF Worker CLI

set -e

echo "📦 Configuration d'OCF Worker CLI..."

# Add /usr/bin to PATH if not already there (should be by default)
if ! echo "$PATH" | grep -q "/usr/bin"; then
    echo "⚠️  /usr/bin n'est pas dans le PATH, cela peut causer des problèmes"
fi

# Test if the binary is accessible
if command -v ocf-worker-cli >/dev/null 2>&1; then
    echo "✅ OCF Worker CLI installé avec succès"
    echo "📍 Version: $(ocf-worker-cli --version 2>/dev/null | head -n1 || echo 'Version inconnue')"
else
    echo "❌ Erreur: OCF Worker CLI n'est pas accessible"
    exit 1
fi

# Reload bash completion if bash-completion is installed
if [ -f /usr/share/bash-completion/bash_completion ]; then
    echo "🔄 Rechargement de l'autocomplétion Bash..."
    # Note: This only affects new shell sessions
fi

# Reload zsh completion if zsh is installed
if command -v zsh >/dev/null 2>&1; then
    echo "🔄 Configuration de l'autocomplétion Zsh..."
    # The completion file is already in the right place
fi

echo ""
echo "🎉 Installation terminée!"
echo ""
echo "Pour commencer:"
echo "  ocf-worker-cli --help"
echo "  ocf-worker-cli health"
echo ""
echo "Pour l'autocomplétion, redémarrez votre shell ou exécutez:"
echo "  # Bash:"
echo "  source /usr/share/bash-completion/completions/ocf-worker-cli"
echo "  # Zsh:"
echo "  autoload -U compinit && compinit"
echo ""