package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// completionCmd représente la commande completion
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Génère les scripts d'autocomplétion pour votre shell",
	Long: `Pour charger l'autocomplétion:

Bash:
  $ source <(ocf-worker-cli completion bash)

  # Pour la charger automatiquement à chaque ouverture de shell:
  ## Linux:
  $ ocf-worker-cli completion bash > /etc/bash_completion.d/ocf-worker-cli
  ## macOS:
  $ ocf-worker-cli completion bash > $(brew --prefix)/etc/bash_completion.d/ocf-worker-cli

Zsh:
  # Si l'autocomplétion shell n'est pas déjà activée dans votre environnement,
  # vous devrez d'abord l'activer. Vous pouvez exécuter la commande suivante une fois:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # Pour charger l'autocomplétion à chaque session:
  ## Linux:
  $ ocf-worker-cli completion zsh > "${fpath[1]}/_ocf-worker-cli"
  ## macOS:
  $ ocf-worker-cli completion zsh > $(brew --prefix)/share/zsh/site-functions/_ocf-worker-cli

  # Vous devrez commencer un nouveau shell pour que cela prenne effet.

Fish:
  $ ocf-worker-cli completion fish | source

  # Pour la charger automatiquement à chaque ouverture de shell:
  $ ocf-worker-cli completion fish > ~/.config/fish/completions/ocf-worker-cli.fish

PowerShell:
  PS> ocf-worker-cli completion powershell | Out-String | Invoke-Expression

  # Pour la charger automatiquement à chaque ouverture de shell:
  PS> ocf-worker-cli completion powershell > ocf-worker-cli.ps1
  # et ajoutez une ligne à votre profil PowerShell pour sourcer ce fichier.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
