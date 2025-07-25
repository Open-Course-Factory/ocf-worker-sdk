package cli

import (
	"context"
	"fmt"
	ocfworker "ocf-worker-sdk"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
)

// jobsCmd représente la commande jobs
var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Gestion des jobs de génération",
	Long: `Commandes pour gérer et surveiller les jobs de génération de présentations.

Exemples:
  ocf-cli jobs list
  ocf-cli jobs status <job-id>
  ocf-cli jobs logs <job-id>`,
}

// jobsListCmd liste les jobs
var jobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Liste les jobs de génération",
	Long:  "Affiche la liste des jobs de génération avec leur statut.",
	RunE:  runJobsList,
}

var (
	jobsListStatus   string
	jobsListCourseID string
	jobsListLimit    int
	jobsListOffset   int
)

func runJobsList(cmd *cobra.Command, args []string) error {
	client := createClient()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := &ocfworker.ListJobsOptions{
		Status:   jobsListStatus,
		CourseID: jobsListCourseID,
		Limit:    jobsListLimit,
		Offset:   jobsListOffset,
	}

	jobs, err := client.Jobs.List(ctx, opts)
	if err != nil {
		return fmt.Errorf("impossible de récupérer les jobs: %w", err)
	}

	cmd.Printf("📋 Jobs de génération\n")
	cmd.Printf("====================\n")

	if jobs.TotalCount == 0 {
		cmd.Printf("Aucun job trouvé.\n")
		return nil
	}

	// En-têtes
	cmd.Printf("%-36s %-12s %-20s %-15s\n", "JOB ID", "STATUS", "CREATED", "COURSE ID")
	cmd.Printf("%s\n", strings.Repeat("-", 85))

	// Afficher les jobs
	for _, job := range jobs.Jobs {
		status := getJobStatusEmoji(job.Status) + string(job.Status)
		createdAt := job.CreatedAt.Format("2006-01-02 15:04")
		courseID := job.CourseID.String()[:8] + "..."

		cmd.Printf("%-36s %-12s %-20s %-15s\n",
			job.ID.String(),
			status,
			createdAt,
			courseID)
	}

	cmd.Printf("\nTotal: %d jobs (page %d, taille: %d)\n",
		jobs.TotalCount, jobs.Page, jobs.PageSize)

	return nil
}

func getJobStatusEmoji(status models.JobStatus) string {
	switch status {
	case models.StatusCompleted:
		return "✅ "
	case models.StatusProcessing:
		return "🔄 "
	case models.StatusPending:
		return "⏳ "
	case models.StatusFailed:
		return "❌ "
	case models.StatusTimeout:
		return "⏰ "
	default:
		return "❓ "
	}
}

// jobsStatusCmd affiche le statut d'un job
var jobsStatusCmd = &cobra.Command{
	Use:   "status [JOB_ID]",
	Short: "Affiche le statut d'un job",
	Long: `Affiche les informations détaillées et le statut actuel d'un job.

Exemples:
  ocf-cli jobs status 550e8400-e29b-41d4-a716-446655440001`,
	Args: cobra.ExactArgs(1),
	RunE: runJobsStatus,
}

func runJobsStatus(cmd *cobra.Command, args []string) error {
	jobID := args[0]
	client := createClient()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	job, err := client.Jobs.Get(ctx, jobID)
	if err != nil {
		return fmt.Errorf("impossible de récupérer le job: %w", err)
	}

	cmd.Printf("📋 Détails du job\n")
	cmd.Printf("=================\n")
	cmd.Printf("ID: %s\n", job.ID)
	cmd.Printf("Course ID: %s\n", job.CourseID)
	cmd.Printf("Status: %s%s\n", getJobStatusEmoji(job.Status), job.Status)
	cmd.Printf("Créé: %s\n", job.CreatedAt.Format("2006-01-02 15:04:05"))
	cmd.Printf("Mis à jour: %s\n", job.UpdatedAt.Format("2006-01-02 15:04:05"))

	if job.StartedAt != nil {
		cmd.Printf("Démarré: %s\n", job.StartedAt.Format("2006-01-02 15:04:05"))
	}

	if job.CompletedAt != nil {
		cmd.Printf("Terminé: %s\n", job.CompletedAt.Format("2006-01-02 15:04:05"))

		if job.StartedAt != nil {
			duration := job.CompletedAt.Sub(*job.StartedAt)
			cmd.Printf("Durée: %s\n", duration.Round(time.Second))
		}
	}

	if job.SourcePath != "" {
		cmd.Printf("Source: %s\n", job.SourcePath)
	}

	if job.ResultPath != "" {
		cmd.Printf("Résultat: %s\n", job.ResultPath)
	}

	if job.Progress > 0 {
		cmd.Printf("Progrès: %d%%\n", job.Progress)
	}

	if job.Error != "" {
		cmd.Printf("\n❌ Erreur:\n%s\n", job.Error)
	}

	if len(job.Logs) > 0 && viper.GetBool("verbose") {
		cmd.Printf("\n📝 Logs:\n")
		for _, log := range job.Logs {
			cmd.Printf("  %s\n", log)
		}
	}

	// Afficher les métadonnées si disponibles
	if len(job.Metadata) > 0 && viper.GetBool("verbose") {
		cmd.Printf("\n📊 Métadonnées:\n")
		for key, value := range job.Metadata {
			cmd.Printf("  %s: %v\n", key, value)
		}
	}

	return nil
}

// jobsLogsCmd affiche les logs d'un job
var jobsLogsCmd = &cobra.Command{
	Use:   "logs [JOB_ID]",
	Short: "Affiche les logs d'un job",
	Long: `Affiche les logs détaillés d'exécution d'un job.

Exemples:
  ocf-cli jobs logs 550e8400-e29b-41d4-a716-446655440001`,
	Args: cobra.ExactArgs(1),
	RunE: runJobsLogs,
}

func runJobsLogs(cmd *cobra.Command, args []string) error {
	jobID := args[0]
	client := createClient()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logs, err := client.Storage.GetLogs(ctx, jobID)
	if err != nil {
		return fmt.Errorf("impossible de récupérer les logs: %w", err)
	}

	cmd.Printf("📝 Logs du job %s\n", jobID)
	cmd.Printf("===============================\n")

	if strings.TrimSpace(logs) == "" {
		cmd.Printf("Aucun log disponible.\n")
		return nil
	}

	cmd.Print(logs)
	return nil
}

func init() {
	rootCmd.AddCommand(jobsCmd)

	// Sous-commandes
	jobsCmd.AddCommand(jobsListCmd)
	jobsCmd.AddCommand(jobsStatusCmd)
	jobsCmd.AddCommand(jobsLogsCmd)

	// Flags pour list
	jobsListCmd.Flags().StringVar(&jobsListStatus, "status", "", "filtrer par statut (pending, processing, completed, failed, timeout)")
	jobsListCmd.Flags().StringVar(&jobsListCourseID, "course-id", "", "filtrer par ID de cours")
	jobsListCmd.Flags().IntVar(&jobsListLimit, "limit", 20, "nombre maximum de résultats")
	jobsListCmd.Flags().IntVar(&jobsListOffset, "offset", 0, "décalage pour la pagination")

	// Aliases
	jobsListCmd.Aliases = []string{"ls", "l"}
	jobsStatusCmd.Aliases = []string{"get", "show"}
	jobsLogsCmd.Aliases = []string{"log"}
}
