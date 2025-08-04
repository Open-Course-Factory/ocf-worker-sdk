package cli

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"ocf-worker-sdk/pkg/generator"
)

var (
	outputDir    string
	subfolder    string
	waitTimeout  time.Duration
	waitInterval time.Duration
	openResult   bool
	npmPackages  []string
)

// generateCmd représente la commande generate
var generateCmd = &cobra.Command{
	Use:   "generate [URL_GITHUB]",
	Short: "Génère une présentation Slidev depuis un dépôt GitHub",
	Long: `Génère une présentation Slidev à partir d'un dépôt GitHub.

Exemples:
  # Génération basique
  ocf-worker-cli generate https://github.com/ttamoud/presentation

  # Avec sous-dossier spécifique
  ocf-worker-cli generate https://github.com/user/repo --subfolder presentations/my-talk

  # Avec options avancées
  ocf-worker-cli generate https://github.com/user/repo \
    --output ./my-presentation \
    --wait-timeout 20m \
    --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error {
	githubURL := args[0]

	// Créer la configuration
	config := &generator.Config{
		GitHubURL:    githubURL,
		APIBaseURL:   viper.GetString("api-url"),
		OutputDir:    outputDir,
		Subfolder:    subfolder,
		Timeout:      viper.GetDuration("timeout"),
		WaitTimeout:  waitTimeout,
		WaitInterval: waitInterval,
		Verbose:      viper.GetBool("verbose"),
		NpmPackages:  npmPackages,
	}

	// Valider la configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration invalide: %w", err)
	}

	// Créer le générateur
	gen := generator.New(config)

	// Contexte avec timeout global
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Lancer la génération
	result, err := gen.Generate(ctx)
	if err != nil {
		return fmt.Errorf("erreur de génération: %w", err)
	}

	// Afficher les résultats
	cmd.Printf("🎉 Génération terminée avec succès!\n")
	cmd.Printf("📁 Sortie: %s\n", result.OutputDir)
	cmd.Printf("🌐 Présentation: %s\n", result.IndexPath)

	// Ouvrir automatiquement si demandé
	if openResult {
		if err := openInBrowser(result.IndexPath); err != nil {
			cmd.Printf("⚠️ Impossible d'ouvrir automatiquement: %v\n", err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// Flags spécifiques à generate
	generateCmd.Flags().StringVarP(&outputDir, "output", "o", "./output", "répertoire de sortie")
	generateCmd.Flags().StringVar(&subfolder, "subfolder", "", "sous-dossier spécifique dans le dépôt")
	generateCmd.Flags().DurationVar(&waitTimeout, "wait-timeout", 15*time.Minute, "timeout d'attente de completion")
	generateCmd.Flags().DurationVar(&waitInterval, "wait-interval", 5*time.Second, "intervalle de polling")
	generateCmd.Flags().BoolVar(&openResult, "open", false, "ouvrir automatiquement la présentation")
	generateCmd.Flags().StringArrayVar(&npmPackages, "npm-package", []string{}, "package npm à installer en plus (peut être utilisé plusieurs fois)")

	// Aliases
	generateCmd.Aliases = []string{"gen", "g"}
}

func openInBrowser(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("file://%s", absPath)

	// Détection de l'OS et commande appropriée
	var cmd string
	switch {
	case isCommand("open"): // macOS
		cmd = "open"
	case isCommand("xdg-open"): // Linux
		cmd = "xdg-open"
	case isCommand("cmd"): // Windows
		cmd = "cmd"
		url = "/c start " + url
	default:
		return fmt.Errorf("aucune commande d'ouverture trouvée")
	}

	return exec.Command(cmd, url).Start()
}

func isCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
