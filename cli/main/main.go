package main

import (
	"runtime/debug"

	"ocf-worker-sdk/cli"
)

// Variables injectées par GoReleaser au moment du build
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	// Injecter les informations de version dans le CLI
	cli.SetVersionInfo(buildVersionInfo())

	// Exécuter le CLI
	cli.Execute()
}

// buildVersionInfo construit les informations de version complètes
func buildVersionInfo() cli.VersionInfo {
	// Si on est en mode développement, essayer de récupérer les infos depuis Git
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			// Essayer de trouver le commit et les informations de build
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					if commit == "none" {
						// Prendre seulement les premiers 7 caractères du commit
						if len(setting.Value) > 7 {
							commit = setting.Value[:7]
						} else {
							commit = setting.Value
						}
					}
				case "vcs.time":
					if date == "unknown" {
						date = setting.Value
					}
				}
			}
		}
	}

	return cli.VersionInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
		BuiltBy: builtBy,
	}
}
