package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// VersionInfo contient les informations de version
type VersionInfo struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}

var versionInfo VersionInfo

// SetVersionInfo définit les informations de version
func SetVersionInfo(info VersionInfo) {
	versionInfo = info
	// Mettre à jour la version de la commande racine
	rootCmd.Version = info.Version
}

// versionCmd représente la commande version
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Affiche les informations de version",
	Long:  `Affiche les informations détaillées de version d'OCF Worker CLI.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("OCF Worker CLI %s\n", versionInfo.Version)

		if cmd.Flag("detailed").Changed {
			fmt.Printf("Version:    %s\n", versionInfo.Version)
			fmt.Printf("Commit:     %s\n", versionInfo.Commit)
			fmt.Printf("Build Date: %s\n", versionInfo.Date)
			fmt.Printf("Built By:   %s\n", versionInfo.BuiltBy)
			fmt.Printf("Go Version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Flag pour les détails
	versionCmd.Flags().BoolP("detailed", "d", false, "affiche les informations détaillées")

	// Aliases
	versionCmd.Aliases = []string{"v", "ver"}
}
