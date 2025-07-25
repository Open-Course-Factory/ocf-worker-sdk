package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// themesCmd représente la commande themes
var themesCmd = &cobra.Command{
	Use:   "themes",
	Short: "Gestion des thèmes Slidev",
	Long: `Commandes pour gérer les thèmes Slidev disponibles et installés.

Exemples:
  ocf-cli themes list
  ocf-cli themes install @slidev/theme-seriph
  ocf-cli themes detect <job-id>`,
}

// themesListCmd liste les thèmes disponibles
var themesListCmd = &cobra.Command{
	Use:   "list",
	Short: "Liste les thèmes Slidev disponibles",
	Long:  "Affiche tous les thèmes Slidev disponibles avec leur statut d'installation.",
	RunE:  runThemesList,
}

func runThemesList(cmd *cobra.Command, args []string) error {
	client := createClient()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	themes, err := client.Themes.ListAvailable(ctx)
	if err != nil {
		return fmt.Errorf("impossible de récupérer les thèmes: %w", err)
	}

	cmd.Printf("🎨 Thèmes Slidev disponibles\n")
	cmd.Printf("============================\n")

	if themes.Count == 0 {
		cmd.Printf("Aucun thème disponible.\n")
		return nil
	}

	// Calculer la largeur des colonnes
	maxNameWidth := 0
	maxVersionWidth := 0
	for _, theme := range themes.Themes {
		if len(theme.Name) > maxNameWidth {
			maxNameWidth = len(theme.Name)
		}
		if len(theme.Version) > maxVersionWidth {
			maxVersionWidth = len(theme.Version)
		}
	}

	// Afficher les thèmes
	for _, theme := range themes.Themes {
		status := "❌"
		if theme.Installed {
			status = "✅"
		}

		cmd.Printf("%s %-*s %-*s %s\n",
			status,
			maxNameWidth, theme.Name,
			maxVersionWidth, theme.Version,
			theme.Description)
	}

	cmd.Printf("\nTotal: %d thèmes (%d installés, %d disponibles)\n",
		themes.Summary.Total,
		themes.Summary.Installed,
		themes.Summary.Available)

	return nil
}

// themesInstallCmd installe un thème
var themesInstallCmd = &cobra.Command{
	Use:   "install [THEME_NAME]",
	Short: "Installe un thème Slidev",
	Long: `Installe un thème Slidev spécifique.

Exemples:
  ocf-cli themes install @slidev/theme-seriph
  ocf-cli themes install academic`,
	Args: cobra.ExactArgs(1),
	RunE: runThemesInstall,
}

func runThemesInstall(cmd *cobra.Command, args []string) error {
	themeName := args[0]
	client := createClient()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd.Printf("🎨 Installation du thème: %s\n", themeName)

	result, err := client.Themes.Install(ctx, themeName)
	if err != nil {
		return fmt.Errorf("erreur d'installation: %w", err)
	}

	if result.Success {
		cmd.Printf("✅ Thème '%s' installé avec succès\n", result.Theme)
	} else {
		cmd.Printf("❌ Échec de l'installation: %s\n", result.Message)
		if result.Error != "" {
			cmd.Printf("Erreur: %s\n", result.Error)
		}
	}

	return nil
}

// themesDetectCmd détecte les thèmes requis pour un job
var themesDetectCmd = &cobra.Command{
	Use:   "detect [JOB_ID]",
	Short: "Détecte les thèmes requis pour un job",
	Long: `Analyse les fichiers sources d'un job pour détecter automatiquement 
les thèmes Slidev requis.

Exemples:
  ocf-cli themes detect 550e8400-e29b-41d4-a716-446655440001`,
	Args: cobra.ExactArgs(1),
	RunE: runThemesDetect,
}

func runThemesDetect(cmd *cobra.Command, args []string) error {
	jobID := args[0]
	client := createClient()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd.Printf("🔍 Détection des thèmes pour le job: %s\n", jobID)

	result, err := client.Themes.DetectForJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("erreur de détection: %w", err)
	}

	cmd.Printf("📊 Résultats de la détection\n")
	cmd.Printf("============================\n")
	cmd.Printf("Thèmes détectés: %d\n", result.DetectedCount)

	if len(result.InstalledThemes) > 0 {
		cmd.Printf("\n✅ Thèmes déjà installés:\n")
		for _, theme := range result.InstalledThemes {
			cmd.Printf("  - %s (%s)\n", theme.Name, theme.Version)
		}
	}

	if len(result.MissingThemes) > 0 {
		cmd.Printf("\n❌ Thèmes manquants:\n")
		for _, theme := range result.MissingThemes {
			cmd.Printf("  - %s\n", theme)
		}

		cmd.Printf("\n💡 Pour installer automatiquement:\n")
		cmd.Printf("   ocf-cli themes auto-install %s\n", jobID)
	}

	return nil
}

// themesAutoInstallCmd installe automatiquement les thèmes manquants
var themesAutoInstallCmd = &cobra.Command{
	Use:   "auto-install [JOB_ID]",
	Short: "Installe automatiquement les thèmes manquants pour un job",
	Long: `Détecte et installe automatiquement tous les thèmes requis pour un job.

Exemples:
  ocf-cli themes auto-install 550e8400-e29b-41d4-a716-446655440001`,
	Args: cobra.ExactArgs(1),
	RunE: runThemesAutoInstall,
}

func runThemesAutoInstall(cmd *cobra.Command, args []string) error {
	jobID := args[0]
	client := createClient()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd.Printf("🎨 Installation automatique des thèmes pour le job: %s\n", jobID)

	result, err := client.Themes.AutoInstallForJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("erreur d'installation automatique: %w", err)
	}

	cmd.Printf("📊 Résultats de l'installation\n")
	cmd.Printf("==============================\n")
	cmd.Printf("Thèmes traités: %d\n", result.TotalThemes)
	cmd.Printf("Succès: %d\n", result.Successful)
	cmd.Printf("Échecs: %d\n", result.Failed)

	if len(result.Results) > 0 {
		cmd.Printf("\nDétails:\n")
		for _, themeResult := range result.Results {
			status := "❌"
			if themeResult.Success {
				status = "✅"
			}

			cmd.Printf("  %s %s", status, themeResult.Theme)
			cmd.Printf(" (%d)", themeResult.Duration)

			cmd.Printf("\n")

			if !themeResult.Success && themeResult.Error != "" {
				cmd.Printf("    Erreur: %s\n", themeResult.Error)
			}

			if viper.GetBool("verbose") && len(themeResult.Logs) > 0 {
				for _, log := range themeResult.Logs {
					cmd.Printf("    Log: %s\n", strings.TrimSpace(log))
				}
			}
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(themesCmd)

	// Sous-commandes
	themesCmd.AddCommand(themesListCmd)
	themesCmd.AddCommand(themesInstallCmd)
	themesCmd.AddCommand(themesDetectCmd)
	themesCmd.AddCommand(themesAutoInstallCmd)

	// Aliases
	themesListCmd.Aliases = []string{"ls", "l"}
	themesInstallCmd.Aliases = []string{"add", "i"}
	themesAutoInstallCmd.Aliases = []string{"auto", "ai"}
}
