package generator

// Result contient les résultats de la génération
type Result struct {
	JobID       string
	CourseID    string
	OutputDir   string
	IndexPath   string
	ArchivePath string
	Files       []string
}
