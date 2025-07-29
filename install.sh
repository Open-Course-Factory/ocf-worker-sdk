#!/bin/bash
# OCF Worker CLI - Installation universelle
# Usage: curl -s https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk/-/raw/main/install.sh | bash

set -e

# Configuration
readonly REPO_URL="https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk"
readonly BINARY_NAME="ocf-worker-cli"
readonly INSTALL_DIR="/usr/local/bin"
readonly COMPLETION_DIR_BASH="/etc/bash_completion.d"
readonly COMPLETION_DIR_ZSH="/usr/local/share/zsh/site-functions"

# Couleurs pour les messages
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Variables globales
FORCE_INSTALL=false
INSTALL_COMPLETION=true
SUDO_REQUIRED=true
DETECTED_OS=""
DETECTED_ARCH=""
VERSION="latest"

# Fonctions utilitaires
log_info() {
    echo -e "${BLUE}ℹ️  ${1}${NC}"
}

log_success() {
    echo -e "${GREEN}✅ ${1}${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  ${1}${NC}"
}

log_error() {
    echo -e "${RED}❌ ${1}${NC}"
    exit 1
}

print_banner() {
    echo -e "${BLUE}"
    cat << 'EOF'
    ____   _____ ______   _      _              _                   _____  _      _____ 
   / __ \ / ____|  ____| | |    | |            | |                 / ____|| |    |_   _|
  | |  | | |    | |__    | |    | |  ___   _ _ | | _   ___  _ _   | |     | |      | |  
  | |  | | |    |  __|   | |    | | / _ \ | '__| |/ / / _ \| '__| | |     | |      | |  
  | |__| | |____| |      | |_/\_| || (_) || |  |   < |  __/| |    | |____ | |____ _| |_ 
   \____/ \_____|_|      |________| \___/ |_|  |_|\_\ \___||_|     \_____||______|_____|

EOF
    echo -e "${NC}"
    echo "🚀 Installation d'OCF Worker CLI"
    echo ""
}

# Détection de l'OS et de l'architecture
detect_platform() {
    log_info "Détection de la plateforme..."
    
    # Détection de l'OS
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        DETECTED_OS="linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        DETECTED_OS="darwin"
    else
        log_error "Système d'exploitation non supporté: $OSTYPE"
    fi
    
    # Détection de l'architecture
    local arch
    arch=$(uname -m)
    case $arch in
        x86_64|amd64)
            DETECTED_ARCH="amd64"
            ;;
        aarch64|arm64)
            DETECTED_ARCH="arm64"
            ;;
        *)
            log_error "Architecture non supportée: $arch"
            ;;
    esac
    
    log_success "Plateforme détectée: ${DETECTED_OS}/${DETECTED_ARCH}"
}

# Vérification des prérequis
check_prerequisites() {
    log_info "Vérification des prérequis..."
    
    # Vérifier curl
    if ! command -v curl &> /dev/null; then
        log_error "curl est requis mais non installé"
    fi
    
    # Vérifier tar
    if ! command -v tar &> /dev/null; then
        log_error "tar est requis mais non installé"
    fi
    
    # Vérifier les permissions d'écriture
    if [[ ! -w "$(dirname "$INSTALL_DIR")" ]] && [[ $EUID -ne 0 ]]; then
        SUDO_REQUIRED=true
        log_warning "Installation en tant que root requise pour $INSTALL_DIR"
    else
        SUDO_REQUIRED=false
    fi
    
    log_success "Prérequis validés"
}

# Récupération de la dernière version
get_latest_version() {
    if [[ "$VERSION" == "latest" ]]; then
        log_info "Récupération de la dernière version..."

        # Essayer plusieurs méthodes pour récupérer les tags
        local api_urls=(
            "${REPO_URL}/-/refs/tags?format=json"
            "${REPO_URL}/-/tags?format=json"
            "${REPO_URL}/tags?format=json"
        )

        for api_url in "${api_urls[@]}"; do
            log_info "Tentative de récupération depuis: $api_url"
            
            # Utiliser curl avec plus d'options pour contourner les redirections
            VERSION=$(curl -L -s -H "Accept: application/json" -H "User-Agent: OCF-Installer/1.0" "$api_url" 2>/dev/null | \
                grep -o '"name":"v[^"]*"' | head -1 | sed 's/"name":"v\([^"]*\)"/\1/' || echo "")
            
            if [[ -n "$VERSION" ]]; then
                log_success "Version détectée: v$VERSION"
                return
            fi
        done

        # Si toutes les tentatives échouent, essayer avec l'API publique GitLab
        log_info "Tentative avec l'API GitLab publique..."
        local project_path="open-course-factory%2Focf-worker-sdk"
        local gitlab_api="https://usine.solution-libre.fr/api/v4/projects/${project_path}/repository/tags"
        
        VERSION=$(curl -L -s -H "Accept: application/json" "$gitlab_api" 2>/dev/null | \
            grep -o '"name":"v[^"]*"' | head -1 | sed 's/"name":"v\([^"]*\)"/\1/' || echo "")

        if [[ -n "$VERSION" ]]; then
            log_success "Version détectée via API: v$VERSION"
        else
            log_warning "Impossible de détecter la version automatiquement"
            log_info "Utilisation de la version 'main' par défaut"
            VERSION="main"
        fi
    fi
}

# Construction de l'URL de téléchargement
build_download_url() {
    local filename="${BINARY_NAME}_${DETECTED_OS^}_"
    
    if [[ "$DETECTED_ARCH" == "amd64" ]]; then
        filename="${filename}x86_64"
    else
        filename="${filename}${DETECTED_ARCH}"
    fi
    
    filename="${filename}.tar.gz"
    
    if [[ "$VERSION" == "main" ]]; then
        # Pour la version de développement, utiliser les artifacts de la dernière pipeline
        DOWNLOAD_URL="${REPO_URL}/-/jobs/artifacts/main/raw/dist/${filename}?job=release"
    else
        # Pour les releases taggées
        DOWNLOAD_URL="${REPO_URL}/-/releases/v${VERSION}/downloads/${filename}"
    fi
    
    log_info "URL de téléchargement: $DOWNLOAD_URL"
}

# Téléchargement et installation
download_and_install() {
    log_info "Téléchargement d'OCF Worker CLI..."
    
    local temp_dir archive_path
    temp_dir=$(mktemp -d)
    archive_path="$temp_dir/ocf-worker-cli.tar.gz"
    
    # Télécharger l'archive
    if ! curl -L -f -s -o "$archive_path" "$DOWNLOAD_URL"; then
        log_error "Échec du téléchargement depuis $DOWNLOAD_URL"
    fi
    
    log_success "Téléchargement terminé"
    
    # Extraction
    log_info "Extraction de l'archive..."
    tar -xzf "$archive_path" -C "$temp_dir"
    
    # Trouver le binaire
    local binary_path
    binary_path=$(find "$temp_dir" -name "$BINARY_NAME" -type f | head -1)
    if [[ -z "$binary_path" ]]; then
        log_error "Binaire $BINARY_NAME non trouvé dans l'archive"
    fi
    
    # Installation du binaire
    log_info "Installation du binaire..."
    if [[ "$SUDO_REQUIRED" == "true" ]]; then
        sudo install -m 755 "$binary_path" "$INSTALL_DIR/"
    else
        install -m 755 "$binary_path" "$INSTALL_DIR/"
    fi
    
    log_success "Binaire installé dans $INSTALL_DIR/$BINARY_NAME"
    
    # Nettoyage
    rm -rf "$temp_dir"
}

# Installation de l'autocomplétion
install_completion() {
    if [[ "$INSTALL_COMPLETION" != "true" ]]; then
        return
    fi
    
    log_info "Installation de l'autocomplétion..."
    
    # Vérifier que le binaire est accessible
    if ! command -v "$BINARY_NAME" &> /dev/null; then
        log_warning "Binaire non accessible, autocomplétion ignorée"
        return
    fi
    
    # Autocomplétion Bash
    if command -v bash &> /dev/null; then
        local bash_completion="$COMPLETION_DIR_BASH/$BINARY_NAME"
        if [[ "$SUDO_REQUIRED" == "true" ]]; then
            sudo mkdir -p "$COMPLETION_DIR_BASH"
            "$BINARY_NAME" completion bash | sudo tee "$bash_completion" > /dev/null 2>&1 || true
        else
            mkdir -p "$COMPLETION_DIR_BASH"
            "$BINARY_NAME" completion bash > "$bash_completion" 2>/dev/null || true
        fi
        
        if [[ -f "$bash_completion" ]]; then
            log_success "Autocomplétion Bash installée"
        fi
    fi
    
    # Autocomplétion Zsh
    if command -v zsh &> /dev/null; then
        local zsh_completion="$COMPLETION_DIR_ZSH/_$BINARY_NAME"
        if [[ "$SUDO_REQUIRED" == "true" ]]; then
            sudo mkdir -p "$COMPLETION_DIR_ZSH"
            "$BINARY_NAME" completion zsh | sudo tee "$zsh_completion" > /dev/null 2>&1 || true
        else
            mkdir -p "$COMPLETION_DIR_ZSH"
            "$BINARY_NAME" completion zsh > "$zsh_completion" 2>/dev/null || true
        fi
        
        if [[ -f "$zsh_completion" ]]; then
            log_success "Autocomplétion Zsh installée"
        fi
    fi
}

# Vérification de l'installation
verify_installation() {
    log_info "Vérification de l'installation..."
    
    if ! command -v "$BINARY_NAME" &> /dev/null; then
        log_error "Installation échouée: binaire non accessible"
    fi
    
    local installed_version
    installed_version=$("$BINARY_NAME" --version 2>/dev/null | head -1 || echo "inconnue")
    
    log_success "Installation réussie!"
    log_info "Version installée: $installed_version"
}

# Messages de fin
print_completion_message() {
    echo ""
    log_success "🎉 OCF Worker CLI installé avec succès!"
    echo ""
    echo "📚 Commandes utiles:"
    echo "  $BINARY_NAME --help                    # Aide générale"
    echo "  $BINARY_NAME health                    # Vérifier la connexion"
    echo "  $BINARY_NAME generate <github-url>     # Générer une présentation"
    echo ""
    echo "🔧 Autocomplétion:"
    echo "  Redémarrez votre shell ou exécutez:"
    echo "  source /etc/bash_completion.d/$BINARY_NAME  # Bash"
    echo "  autoload -U compinit && compinit              # Zsh"
    echo ""
    echo "📖 Documentation complète:"
    echo "  $REPO_URL"
    echo ""
}

# Gestion des arguments de ligne de commande
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                FORCE_INSTALL=true
                shift
                ;;
            --no-completion)
                INSTALL_COMPLETION=false
                shift
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --help)
                print_help
                exit 0
                ;;
            *)
                log_error "Option inconnue: $1"
                ;;
        esac
    done
}

print_help() {
    cat << EOF
OCF Worker CLI - Script d'installation

Usage: curl -s https://usine.solution-libre.fr/.../install.sh | bash

Options:
  --force            Force la réinstallation même si déjà installé
  --no-completion    Ne pas installer l'autocomplétion
  --version VERSION  Installer une version spécifique (défaut: latest)
  --help             Afficher cette aide

Exemples:
  # Installation standard
  curl -s https://usine.solution-libre.fr/.../install.sh | bash
  
  # Installation d'une version spécifique
  curl -s https://usine.solution-libre.fr/.../install.sh | bash -s -- --version 0.1.0
  
  # Installation sans autocomplétion
  curl -s https://usine.solution-libre.fr/.../install.sh | bash -s -- --no-completion

EOF
}

# Vérification si déjà installé
check_existing_installation() {
    if command -v "$BINARY_NAME" &> /dev/null && [[ "$FORCE_INSTALL" != "true" ]]; then
        local current_version
        current_version=$("$BINARY_NAME" --version 2>/dev/null | head -1 || echo "inconnue")
        
        log_warning "$BINARY_NAME est déjà installé (version: $current_version)"
        log_info "Utilisez --force pour forcer la réinstallation"
        exit 0
    fi
}

# Fonction principale
main() {
    parse_args "$@"
    
    print_banner
    check_existing_installation
    detect_platform
    check_prerequisites
    get_latest_version
    build_download_url
    download_and_install
    install_completion
    verify_installation
    print_completion_message
}

# Point d'entrée
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi