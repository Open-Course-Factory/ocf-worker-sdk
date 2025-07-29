#!/bin/bash
# Post-installation script for OCF Worker CLI

set -e

echo "📦 Configuration d'OCF Worker CLI..."

# Test if the binary is accessible
if command -v ocf-worker-cli >/dev/null 2>&1; then
    echo "✅ OCF Worker CLI installé avec succès"
    echo "📍 Version: $(ocf-worker-cli --version 2>/dev/null | head -n1 || echo 'Version inconnue')"
else
    echo "❌ Erreur: OCF Worker CLI n'est pas accessible"
    exit 1
fi

# Configuration de l'autocomplétion
echo "🔄 Configuration de l'autocomplétion..."

# Bash completion
if [ -f /usr/share/bash-completion/completions/ocf-worker-cli ]; then
    echo "✅ Autocomplétion Bash installée"
    # Pas besoin de recharger ici, elle sera active au prochain démarrage de shell
else
    echo "⚠️ Fichier d'autocomplétion Bash manquant"
fi

# Zsh completion
if [ -f /usr/share/zsh/vendor-completions/_ocf-worker-cli ]; then
    echo "✅ Autocomplétion Zsh installée"
else
    echo "⚠️ Fichier d'autocomplétion Zsh manquant"
fi

echo ""
echo "🎉 Installation terminée!"
echo ""
echo "Pour commencer:"
echo "  ocf-worker-cli --help"
echo "  ocf-worker-cli health"
echo ""
echo "🔧 Pour activer l'autocomplétion dans votre shell actuel:"
echo "  # Bash:"
echo "  source /usr/share/bash-completion/completions/ocf-worker-cli"
echo "  # Zsh:"
echo "  autoload -U compinit && compinit"
echo ""
echo "💡 L'autocomplétion sera automatiquement active dans les nouveaux shells."
echo ""