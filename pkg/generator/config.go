package generator

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Config contient la configuration pour la génération
type Config struct {
	GitHubURL    string
	APIBaseURL   string
	AuthToken    string
	OutputDir    string
	Subfolder    string
	Timeout      time.Duration
	WaitTimeout  time.Duration
	WaitInterval time.Duration
	Verbose      bool
	NpmPackages  []string
}

// Validate valide la configuration
func (c *Config) Validate() error {
	if c.GitHubURL == "" {
		return fmt.Errorf("URL GitHub requise")
	}

	// Valider l'URL GitHub
	if !strings.HasPrefix(c.GitHubURL, "https://github.com/") {
		return fmt.Errorf("URL GitHub invalide: doit commencer par https://github.com/")
	}

	// Valider l'URL de l'API
	if _, err := url.Parse(c.APIBaseURL); err != nil {
		return fmt.Errorf("URL API invalide: %w", err)
	}

	// Valeurs par défaut
	if c.OutputDir == "" {
		c.OutputDir = "./output"
	}

	if c.Timeout == 0 {
		c.Timeout = 60 * time.Second
	}

	if c.WaitTimeout == 0 {
		c.WaitTimeout = 15 * time.Minute
	}

	if c.WaitInterval == 0 {
		c.WaitInterval = 5 * time.Second
	}

	return nil
}
