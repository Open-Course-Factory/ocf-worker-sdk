.PHONY: build install test clean run help completion goreleaser-check goreleaser-build goreleaser-release

BINARY_NAME := ocf-worker-cli
MAIN_PACKAGE := ./cli/main
BUILD_FLAGS := -ldflags="-s -w"

# Version info (sera overridé par GoReleaser)
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILT_BY ?= makefile

# Enhanced build flags avec version info
ENHANCED_BUILD_FLAGS := -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -X main.builtBy=$(BUILT_BY)"


help: ## Affiche cette aide
	@echo "OCF Worker CLI"
	@echo "=============="
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

install: ## Installe les dépendances
	@echo "📦 Installation des dépendances..."
	go mod download
	go mod tidy

build: ## Compile le CLI
	@echo "🔨 Compilation d'OCF Worker CLI..."
	go build $(ENHANCED_BUILD_FLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✅ Binaire créé: $(BINARY_NAME)"

install-binary: build ## Installe le binaire dans $GOPATH/bin
	@echo "📦 Installation du binaire..."
	go install $(ENHANCED_BUILD_FLAGS) $(MAIN_PACKAGE)
	@echo "✅ ocf-worker-cli installé dans $GOPATH/bin"

test: ## Lance les tests
	@echo "🧪 Tests en cours..."
	go test -v ./...

test-coverage: ## Tests avec couverture
	@echo "🧪 Tests avec couverture..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean: ## Nettoie les fichiers générés
	@echo "🧹 Nettoyage..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf ./output

# === Packaging ===

completion: build ## Génère les fichiers d'autocomplétion
	@echo "🔧 Génération de l'autocomplétion..."
	@mkdir -p scripts/completion
	@./$(BINARY_NAME) completion bash > scripts/completion/bash_completion
	@./$(BINARY_NAME) completion zsh > scripts/completion/zsh_completion
	@./$(BINARY_NAME) completion fish > scripts/completion/fish_completion
	@echo "✅ Fichiers d'autocomplétion générés"

goreleaser-check: ## Vérifie la configuration GoReleaser
	@echo "🔍 Vérification de la configuration GoReleaser..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "❌ GoReleaser n'est pas installé. Installez-le avec:"; \
		echo "   curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh"; \
		exit 1; \
	fi
	goreleaser check

goreleaser-build: goreleaser-check completion ## Build de test avec GoRelealer (sans release)
	@echo "🔨 Build de test avec GoReleaser..."
	goreleaser build --clean --snapshot

goreleaser-snapshot: goreleaser-check completion ## Build complet (binaires + packages) en mode snapshot
	@echo "🔨 Build complet avec GoReleaser (snapshot)..."
	goreleaser release --clean --snapshot

goreleaser-release: goreleaser-check completion ## Release complète avec GoReleaser (nécessite un tag Git)
	@echo "🚀 Release avec GoReleaser..."
	@if [ -z "$(shell git tag --points-at HEAD)" ]; then \
		echo "❌ Aucun tag Git trouvé sur le commit actuel."; \
		echo "   Créez un tag avec: git tag -a v0.1.0 -m 'Release v0.1.0'"; \
		exit 1; \
	fi
	goreleaser release --clean

# === Tâches de développement et test ===

dev-build: build ## Build rapide pour développement
	@echo "🔄 Build de développement..."

dev-test: build ## Test rapide avec le binaire local
	@echo "🧪 Test du binaire local..."
	@./$(BINARY_NAME) --version
	@./$(BINARY_NAME) --help

# Exemples d'utilisation

run-help: build ## Affiche l'aide du CLI
	./$(BINARY_NAME) --help

run-health: build ## Vérifie la santé d'OCF Worker
	./$(BINARY_NAME) health

run-themes: build ## Liste les thèmes disponibles
	./$(BINARY_NAME) themes list

run-example: build ## Génère avec exemple personnalisable
	./$(BINARY_NAME) generate \
		"$(URL)" \
		--output "$(OUTPUT)" \
		--verbose

lint: ## Lance le linter
	@echo "🔍 Linting..."
	golangci-lint run

# Utilitaires

check-deps: ## Vérifie les dépendances
	@echo "🔍 Vérification des dépendances..."
	go mod verify
	go mod why -m all

update-deps: ## Met à jour les dépendances
	@echo "📦 Mise à jour des dépendances..."
	go get -u ./...
	go mod tidy

release: clean test goreleaser-build ## Prépare une release
	@echo "🚀 Préparation de la release..."
	@echo "✅ Prêt pour la release"

# === Informations système ===

info: ## Affiche les informations système
	@echo "🔍 Informations système:"
	@echo "Go version: $(shell go version)"
	@echo "Git commit: $(COMMIT)"
	@echo "Build date: $(DATE)"
	@echo "Platform: $(shell go env GOOS)/$(shell go env GOARCH)"
	@if command -v goreleaser >/dev/null 2>&1; then \
		echo "GoReleaser: $(shell goreleaser --version 2>/dev/null | head -n1)"; \
	else \
		echo "GoReleaser: non installé"; \
	fi