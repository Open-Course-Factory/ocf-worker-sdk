.PHONY: build install test clean run help completion goreleaser-check goreleaser-build goreleaser-release
.PHONY: install-script test-install-script deploy-install-script release-complete validate-all

BINARY_NAME := ocf-worker-cli
MAIN_PACKAGE := ./cli/main
BUILD_FLAGS := -ldflags="-s -w"

# Configuration du projet
PROJECT_URL := https://usine.solution-libre.fr/open-course-factory/ocf-worker-sdk
INSTALL_SCRIPT_URL := $(PROJECT_URL)/-/raw/main/install.sh

# Version info (sera overridé par GoReleaser)
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILT_BY ?= makefile

# Enhanced build flags avec version info
ENHANCED_BUILD_FLAGS := -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -X main.builtBy=$(BUILT_BY)"

# Couleurs pour les messages
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
RED := \033[0;31m
NC := \033[0m # No Color

# ==============================
# AIDE ET INFORMATIONS
# ==============================

help: ## Affiche cette aide
	@echo "$(BLUE)OCF Worker CLI - Build & Distribution$(NC)"
	@echo "======================================"
	@echo ""
	@echo "$(YELLOW)🔨 Build & Development:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(build|install|test|clean|run)' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)📦 Packaging:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(goreleaser|completion)' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🚀 Distribution:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(install-script|deploy|release)' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)🔍 Utilitaires:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '(info|validate|check)' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

# ==============================
# BUILD & DEVELOPMENT (Section existante améliorée)
# ==============================

install: ## Installe les dépendances
	@echo "$(BLUE)📦 Installation des dépendances...$(NC)"
	go mod download
	go mod tidy
	@echo "$(GREEN)✅ Dépendances installées$(NC)"

build: ## Compile le CLI
	@echo "$(BLUE)🔨 Compilation d'OCF Worker CLI...$(NC)"
	go build $(ENHANCED_BUILD_FLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "$(GREEN)✅ Binaire créé: $(BINARY_NAME)$(NC)"

install-binary: build ## Installe le binaire dans $GOPATH/bin
	@echo "$(BLUE)📦 Installation du binaire...$(NC)"
	go install $(ENHANCED_BUILD_FLAGS) $(MAIN_PACKAGE)
	@echo "$(GREEN)✅ ocf-worker-cli installé dans $$GOPATH/bin$(NC)"

test: ## Lance les tests
	@echo "$(BLUE)🧪 Tests en cours...$(NC)"
	go test -v ./...

test-coverage: ## Tests avec couverture
	@echo "$(BLUE)🧪 Tests avec couverture...$(NC)"
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Rapport de couverture: coverage.html$(NC)"

clean: ## Nettoie les fichiers générés
	@echo "$(BLUE)🧹 Nettoyage...$(NC)"
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf ./dist
	rm -rf scripts/completion
	@echo "$(GREEN)✅ Nettoyage terminé$(NC)"

# ==============================
# PACKAGING (Section améliorée)
# ==============================

completion: build ## Génère les fichiers d'autocomplétion
	@echo "$(BLUE)🔧 Génération de l'autocomplétion...$(NC)"
	@mkdir -p scripts/completion
	@./$(BINARY_NAME) completion bash > scripts/completion/bash_completion
	@./$(BINARY_NAME) completion zsh > scripts/completion/zsh_completion
	@./$(BINARY_NAME) completion fish > scripts/completion/fish_completion
	@echo "$(GREEN)✅ Fichiers d'autocomplétion générés$(NC)"

goreleaser-check: ## Vérifie la configuration GoReleaser
	@echo "$(BLUE)🔍 Vérification de la configuration GoReleaser...$(NC)"
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "$(RED)❌ GoReleaser n'est pas installé$(NC)"; \
		echo "Installation: curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh"; \
		exit 1; \
	fi
	goreleaser check
	@echo "$(GREEN)✅ Configuration GoReleaser valide$(NC)"

goreleaser-build: goreleaser-check completion ## Build de test avec GoReleaser (sans release)
	@echo "$(BLUE)🔨 Build de test avec GoReleaser...$(NC)"
	goreleaser build --clean --snapshot
	@echo "$(GREEN)✅ Build terminé, artifacts dans ./dist/$(NC)"

goreleaser-snapshot: goreleaser-check completion ## Build complet (binaires + packages) en mode snapshot
	@echo "$(BLUE)🔨 Build complet avec GoReleaser (snapshot)...$(NC)"
	goreleaser release --clean --snapshot
	@echo "$(GREEN)✅ Build complet terminé$(NC)"
	@echo "$(YELLOW)📦 Packages disponibles:$(NC)"
	@ls -la dist/*.deb dist/*.tar.gz 2>/dev/null || true

goreleaser-release: goreleaser-check completion ## Release complète avec GoReleaser (nécessite un tag Git)
	@echo "$(BLUE)🚀 Release avec GoReleaser...$(NC)"
	@if [ -z "$(shell git tag --points-at HEAD)" ]; then \
		echo "$(RED)❌ Aucun tag Git trouvé sur le commit actuel$(NC)"; \
		echo "Créez un tag avec: git tag -a v0.1.0 -m 'Release v0.1.0'"; \
		exit 1; \
	fi
	goreleaser release --clean
	@echo "$(GREEN)✅ Release terminée$(NC)"

# ==============================
# DISTRIBUTION & INSTALLATION
# ==============================

install-script: ## Valide le script d'installation
	@echo "$(BLUE)🔍 Validation du script d'installation...$(NC)"
	@if command -v shellcheck >/dev/null 2>&1; then \
		shellcheck install.sh; \
		echo "$(GREEN)✅ Script d'installation validé$(NC)"; \
	else \
		echo "$(YELLOW)⚠️ shellcheck non installé, validation ignorée$(NC)"; \
	fi

test-install-script: install-script ## Test le script d'installation localement
	@echo "$(BLUE)🧪 Test du script d'installation...$(NC)"
	@if [ ! -f "install.sh" ]; then \
		echo "$(RED)❌ Fichier install.sh non trouvé$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Simulation de l'installation (dry-run)...$(NC)"
	@bash -n install.sh
	@echo "$(GREEN)✅ Script d'installation syntaxiquement correct$(NC)"

deploy-install-script: goreleaser-snapshot ## Déploie le script d'installation et les packages
	@echo "$(BLUE)🌐 Préparation du déploiement...$(NC)"
	@echo "$(YELLOW)📋 Instructions de déploiement:$(NC)"
	@echo ""
	@echo "1. Commitez et pushez install.sh:"
	@echo "   git add install.sh && git commit -m 'Add install script' && git push"
	@echo ""
	@echo "2. Créez un tag de release:"
	@echo "   git tag -a v0.1.0 -m 'Release v0.1.0' && git push --tags"
	@echo ""
	@echo "3. Le pipeline CI/CD se chargera automatiquement du déploiement"
	@echo ""
	@echo "$(GREEN)🌐 URL d'installation future:$(NC)"
	@echo "   curl -s $(INSTALL_SCRIPT_URL) | bash"

# ==============================
# RELEASE COMPLÈTE
# ==============================

validate-all: test lint goreleaser-check install-script ## Valide tout avant release
	@echo "$(BLUE)🔍 Validation complète...$(NC)"
	@echo "$(GREEN)✅ Toutes les validations sont passées$(NC)"

pre-release: validate-all ## Prépare une release (validation + build)
	@echo "$(BLUE)🚀 Préparation de la release...$(NC)"
	$(MAKE) goreleaser-snapshot
	@echo ""
	@echo "$(GREEN)✅ Prêt pour la release$(NC)"
	@echo ""
	@echo "$(YELLOW)📋 Prochaines étapes:$(NC)"
	@echo "1. Vérifiez les packages dans ./dist/"
	@echo "2. Testez l'installation: sudo dpkg -i dist/*.deb"
	@echo "3. Créez un tag: git tag -a v0.x.x -m 'Release v0.x.x'"
	@echo "4. Pushez le tag: git push --tags"

release-complete: ## Release complète avec tag automatique (version patch)
	@echo "$(BLUE)🚀 Release complète automatique...$(NC)"
	@if ! git diff --quiet; then \
		echo "$(RED)❌ Changements non commités détectés$(NC)"; \
		exit 1; \
	fi
	@$(MAKE) validate-all
	@# Génération du tag automatique (patch version)
	@LAST_TAG=$$(git tag -l "v*" | sort -V | tail -1); \
	if [ -z "$$LAST_TAG" ]; then \
		NEW_TAG="v0.1.0"; \
	else \
		NEW_TAG=$$(echo $$LAST_TAG | awk -F. '{printf "v%d.%d.%d", $$1, $$2, $$3+1}' | sed 's/v\([0-9]*\)\.\([0-9]*\)\.\([0-9]*\)/v\1.\2.\3/'); \
	fi; \
	echo "$(YELLOW)Création du tag: $$NEW_TAG$(NC)"; \
	git tag -a $$NEW_TAG -m "Automatic release $$NEW_TAG"; \
	echo "$(GREEN)✅ Tag créé: $$NEW_TAG$(NC)"; \
	echo "$(BLUE)Push du tag...$(NC)"; \
	git push --tags; \
	echo "$(GREEN)🎉 Release $$NEW_TAG lancée! Consultez les pipelines GitLab$(NC)"

# ==============================
# DÉVELOPPEMENT & TESTS
# ==============================

dev-build: build ## Build rapide pour développement
	@echo "$(GREEN)🔄 Build de développement terminé$(NC)"

dev-test: build ## Test rapide avec le binaire local
	@echo "$(BLUE)🧪 Test du binaire local...$(NC)"
	@./$(BINARY_NAME) --version
	@./$(BINARY_NAME) --help >/dev/null
	@echo "$(GREEN)✅ Binaire fonctionnel$(NC)"

# Exemples d'utilisation
run-help: build ## Affiche l'aide du CLI
	./$(BINARY_NAME) --help

run-health: build ## Vérifie la santé d'OCF Worker  
	./$(BINARY_NAME) health

run-themes: build ## Liste les thèmes disponibles
	./$(BINARY_NAME) themes list

run-example: build ## Génère avec exemple personnalisable
	@echo "$(YELLOW)Usage: make run-example URL=<github-url> [OUTPUT=<dir>]$(NC)"
	@if [ -z "$(URL)" ]; then \
		echo "$(RED)❌ Variable URL requise$(NC)"; \
		echo "Exemple: make run-example URL=https://github.com/ttamoud/presentation"; \
		exit 1; \
	fi
	./$(BINARY_NAME) generate "$(URL)" --output "$(or $(OUTPUT),./output)" --verbose

# ==============================
# UTILITAIRES & MAINTENANCE
# ==============================

lint: ## Lance le linter
	@echo "$(BLUE)🔍 Linting...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)⚠️ golangci-lint non installé$(NC)"; \
	fi

check-deps: ## Vérifie les dépendances
	@echo "$(BLUE)🔍 Vérification des dépendances...$(NC)"
	go mod verify
	go mod why -m all

update-deps: ## Met à jour les dépendances
	@echo "$(BLUE)📦 Mise à jour des dépendances...$(NC)"
	go get -u ./...
	go mod tidy
	@echo "$(GREEN)✅ Dépendances mises à jour$(NC)"

# Informations système
info: ## Affiche les informations système
	@echo "$(BLUE)🔍 Informations système:$(NC)"
	@echo "Go version: $(shell go version)"
	@echo "Git commit: $(COMMIT)"
	@echo "Build date: $(DATE)"
	@echo "Platform: $(shell go env GOOS)/$(shell go env GOARCH)"
	@echo "Project URL: $(PROJECT_URL)"
	@echo "Install URL: $(INSTALL_SCRIPT_URL)"
	@if command -v goreleaser >/dev/null 2>&1; then \
		echo "GoReleaser: $(shell goreleaser --version 2>/dev/null | head -n1)"; \
	else \
		echo "GoReleaser: non installé"; \
	fi

# Debug des variables
debug: ## Debug des variables Makefile
	@echo "$(BLUE)🐛 Variables de debug:$(NC)"
	@echo "BINARY_NAME = $(BINARY_NAME)"
	@echo "VERSION = $(VERSION)"
	@echo "COMMIT = $(COMMIT)"
	@echo "DATE = $(DATE)"
	@echo "BUILD_FLAGS = $(BUILD_FLAGS)"
	@echo "PROJECT_URL = $(PROJECT_URL)"
	@echo "INSTALL_SCRIPT_URL = $(INSTALL_SCRIPT_URL)"

# ==============================
# INSTALLATION DEPUIS LE WEB
# ==============================

install-from-web: ## Installe depuis le script web (test local)
	@echo "$(BLUE)🌐 Installation depuis le script web...$(NC)"
	@echo "$(YELLOW)⚠️ Ceci installera la version officielle$(NC)"
	@read -p "Continuer? [y/N] " -n 1 -r; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo ""; \
		curl -s $(INSTALL_SCRIPT_URL) | bash; \
	else \
		echo ""; \
		echo "$(YELLOW)Installation annulée$(NC)"; \
	fi