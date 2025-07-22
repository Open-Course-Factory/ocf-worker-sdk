package main

import (
	"context"
	"fmt"
	"log"
	"time"

	ocfworker "ocf-worker-sdk"

	"github.com/Open-Course-Factory/ocf-worker/pkg/models"
	"github.com/google/uuid"
)

func main() {
	// Créer le client avec options
	client := ocfworker.NewClient(
		"http://localhost:8081",
		ocfworker.WithTimeout(60*time.Second),
	)

	ctx := context.Background()

	// 1. Vérifier la santé du service
	health, err := client.Health.Check(ctx)
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Printf("Service status: %s\n", health.Status)

	// 2. Créer un nouveau job
	jobID := uuid.New()
	courseID := uuid.New()

	req := &models.GenerationRequest{
		JobID:      jobID,
		CourseID:   courseID,
		SourcePath: "/sources",
	}

	// Option 1: Polling manuel
	job, err := client.Jobs.Create(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create job: %v", err)
	}
	fmt.Println(job.Logs)

	// Upload des fichiers sources
	files := []ocfworker.FileUpload{
		{
			Name:        "slides.md",
			Content:     []byte("---\ntheme: default\n---\n\n# Ma présentation\n\n---\n\n## Slide 2\nContenu..."),
			ContentType: "text/markdown",
		},
	}

	uploadResp, err := client.Storage.UploadSources(ctx, jobID.String(), files)
	if err != nil {
		log.Fatalf("Failed to upload sources: %v", err)
	}
	fmt.Printf("Uploaded %d files\n", uploadResp.Count)

	// Installer les thèmes automatiquement
	themeResult, err := client.Themes.AutoInstallForJob(ctx, jobID.String())
	if err != nil {
		log.Printf("Theme installation warning: %v", err)
	} else {
		fmt.Printf("Installed %d themes\n", themeResult.Successful)
	}

	// Polling manuel
	for {
		status, err := client.Jobs.Get(ctx, jobID.String())
		if err != nil {
			log.Fatalf("Failed to get job status: %v", err)
		}

		fmt.Printf("Job status: %s\n", status.Status)

		if status.Status == models.StatusCompleted {
			fmt.Println("Job completed successfully!")
			break
		} else if status.Status == models.StatusFailed {
			log.Fatalf("Job failed: %s", status.Error)
		}

		time.Sleep(5 * time.Second)
	}

	// Option 2: Polling automatique (alternative)
	// job, err := client.Jobs.CreateAndWait(ctx, req, &ocfworker.WaitOptions{
	//     Interval: 5 * time.Second,
	//     Timeout:  10 * time.Minute,
	// })

	// 3. Télécharger les résultats
	results, err := client.Storage.ListResults(ctx, courseID.String())
	if err != nil {
		log.Fatalf("Failed to list results: %v", err)
	}

	fmt.Printf("Generated %d result files\n", results.Count)
	for _, file := range results.Files {
		fmt.Printf("- %s\n", file)
	}

	archiveReader, err := client.Archive.DownloadArchive(ctx, courseID.String(), &ocfworker.DownloadArchiveOptions{
		Format:   "zip",
		Compress: &[]bool{true}[0],
	})
	if err != nil {
		log.Printf("Failed to download archive: %v", err)
	} else {
		defer archiveReader.Close()
		fmt.Println("Archive downloaded successfully")
	}
}
