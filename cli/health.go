package cli

import (
	"context"
	"fmt"
	"ocf-worker-sdk/pkg/generator"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	ocfworker "ocf-worker-sdk"
)

// healthCmd représente la commande health
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Vérifie la santé du service OCF Worker",
	Long: `Vérifie l'état de santé du service OCF Worker et affiche des informations détaillées.

Exemples:
  ocf-cli health
  ocf-cli health --verbose`,
	RunE: runHealth,
}

func runHealth(cmd *cobra.Command, args []string) error {
	// Créer le client
	client := createClient()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Vérifier la santé générale
	health, err := client.Health.Check(ctx)
	if err != nil {
		return fmt.Errorf("impossible de vérifier la santé: %w", err)
	}

	// Afficher les résultats
	cmd.Printf("🏥 Santé du service OCF Worker\n")
	cmd.Printf("===============================\n")
	cmd.Printf("Status: %s\n", getStatusEmoji(health.Status)+health.Status)
	cmd.Printf("Service: %s\n", health.Service)
	cmd.Printf("Version: %s\n", health.Version)
	cmd.Printf("Uptime: %s\n", health.Uptime)

	if viper.GetBool("verbose") {
		// Vérifier la santé des workers
		workerHealth, err := client.Worker.Health(ctx)
		if err != nil {
			cmd.Printf("\n⚠️ Impossible de vérifier les workers: %v\n", err)
		} else {
			cmd.Printf("\n🔧 Workers\n")
			cmd.Printf("Status: %s\n", getStatusEmoji(workerHealth.Status)+workerHealth.Status)
			cmd.Printf("Workers actifs: %d/%d\n",
				workerHealth.WorkerPool.ActiveWorkers,
				workerHealth.WorkerPool.WorkerCount)
			cmd.Printf("Queue: %d jobs\n", workerHealth.WorkerPool.QueueSize)
		}

		// Informations sur le stockage
		storageInfo, err := client.Storage.GetStorageInfo(ctx)
		if err != nil {
			cmd.Printf("\n⚠️ Impossible de récupérer les infos de stockage: %v\n", err)
		} else {
			cmd.Printf("\n💾 Stockage\n")
			cmd.Printf("Type: %s\n", storageInfo.StorageType)
			cmd.Printf("Status: %s\n", getStatusEmoji(storageInfo.Status)+storageInfo.Status)

			if storageInfo.Capacity != nil {
				usedGB := float64(storageInfo.Capacity.Used) / (1024 * 1024 * 1024)
				totalGB := float64(storageInfo.Capacity.Total) / (1024 * 1024 * 1024)
				cmd.Printf("Utilisé: %.2f GB / %.2f GB\n",
					usedGB, totalGB)
			}
		}
	}

	return nil
}

func getStatusEmoji(status string) string {
	switch status {
	case "healthy":
		return "✅ "
	case "degraded":
		return "⚠️ "
	case "unhealthy":
		return "❌ "
	default:
		return "❓ "
	}
}

func createClient() *ocfworker.Client {
	opts := []ocfworker.Option{
		ocfworker.WithTimeout(viper.GetDuration("timeout")),
	}

	if token := viper.GetString("token"); token != "" {
		opts = append(opts, ocfworker.WithAuth(token))
	}

	if viper.GetBool("verbose") {
		opts = append(opts, ocfworker.WithLogger(generator.NewVerboseLogger()))
	}

	return ocfworker.NewClient(viper.GetString("api-url"), opts...)
}

func init() {
	rootCmd.AddCommand(healthCmd)
}
