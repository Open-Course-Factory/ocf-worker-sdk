package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Flags globaux
	cfgFile   string
	apiURL    string
	authToken string
	timeout   time.Duration
	verbose   bool
)

// rootCmd représente la commande de base quand appelée sans sous-commandes
var rootCmd = &cobra.Command{
	Use:   "ocf-worker-cli",
	Short: "OCF Worker CLI - Générateur de présentations Slidev",
	Long: `OCF Worker CLI est un outil en ligne de commande pour générer des présentations Slidev
à partir de dépôts GitHub en utilisant l'API OCF Worker.

Exemples:
  ocf-worker-cli generate https://github.com/nekomeowww/talks/tree/main/packages/2024-08-23-kubecon-hk
  ocf-worker-cli health
  ocf-worker-cli themes list`,
	Version: "0.0.1",
}

// Execute ajoute toutes les commandes enfant à la commande racine et définit les flags appropriés.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Flags globaux persistants
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "fichier de config (défaut: $HOME/.ocf-worker-cli.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "http://localhost:8081", "URL de base de l'API OCF Worker")
	rootCmd.PersistentFlags().StringVar(&authToken, "token", "", "token d'authentification OCF Worker")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 60*time.Second, "timeout des requêtes HTTP")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "mode verbeux")

	// Liaison avec viper
	viper.BindPFlag("api-url", rootCmd.PersistentFlags().Lookup("api-url"))
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig lit le fichier de configuration et les variables d'environnement si définies.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".ocf-worker-cli")
	}

	// Variables d'environnement
	viper.SetEnvPrefix("OCF")
	viper.AutomaticEnv()

	// Lire le fichier de config
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintf(os.Stderr, "Utilisation du fichier de config: %s\n", viper.ConfigFileUsed())
	}
}
