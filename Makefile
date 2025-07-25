.PHONY: build install test clean run help

BINARY_NAME := ocf-cli
MAIN_PACKAGE := ./cmd
BUILD_FLAGS := -ldflags="-s -w"

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
	@echo "🔨 Compilation d'OCF CLI..."
	go build $(BUILD_FLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "✅ Binaire créé: $(BINARY_NAME)"

install-binary: build ## Installe le binaire dans $GOPATH/bin
	@echo "📦 Installation du binaire..."
	go install $(BUILD_FLAGS) $(MAIN_PACKAGE)
	@echo "✅ ocf-cli installé dans $GOPATH/bin"

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

# Exemples d'utilisation

run-help: build ## Affiche l'aide du CLI
	./$(BINARY_NAME) --help

run-health: build ## Vérifie la santé d'OCF Worker
	./$(BINARY_NAME) health

run-themes: build ## Liste les thèmes disponibles
	./$(BINARY_NAME) themes list

run-kubecon: build ## Génère la présentation KubeCon HK
	./$(BINARY_NAME) generate \
		"https://github.com/nekomeowww/talks/tree/main/packages/2024-08-23-kubecon-hk" \
		--output "./kubecon-output" \
		--verbose

run-example: build ## Génère avec exemple personnalisable
	./$(BINARY_NAME) generate \
		"$(URL)" \
		--output "$(OUTPUT)" \
		--verbose

dev: ## Mode développement
	@echo "🔄 Mode développement..."
	@go run $(MAIN_PACKAGE) generate \
		"https://github.com/nekomeowww/talks/tree/main/packages/2024-08-23-kubecon-hk" \
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

release: clean test build ## Prépare une release
	@echo "🚀 Préparation de la release..."
	@echo "Binaire prêt: $(BINARY_NAME)"
	@./$(BINARY_NAME) --version