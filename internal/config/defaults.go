package config

import (
	"os"
	"path/filepath"
)

// NewDefaultConfig creates a new Config with default values
func NewDefaultConfig() *Config {
	homedir, _ := os.UserHomeDir()

	return &Config{
		Core: Core{
			Trash: TrashConfig{
				// Default to composite strategy
				Strategy: "auto",
				GomiDir:  filepath.Join(homedir, ".gomi"),
			},
			HomeFallback: true,
			Restore: RestoreConfig{
				Confirm: true,
				Verbose: true,
			},
			PermanentDelete: PermanentDeleteConfig{
				Enable: false,
			},
			Logging: LoggingConfig{
				Enabled: true,
				Level:   "debug",
				Rotation: RotationConfig{
					MaxSize:  "10MB",
					MaxFiles: 3,
				},
			},
		},
		UI: UI{
			Density: "spacious",
			Preview: PreviewConfig{
				SyntaxHighlight:  true,
				Colorscheme:      "nord",
				DirectoryCommand: "ls -GF -1 -A --color=always",
			},
			Paginator: "dots",
			Style: StyleConfig{
				ListView: ListViewConfig{
					IndentOnSelect: true,
					Cursor:         "#AD58B4",
					Selected:       "#5FB458",
				},
				DetailView: DetailViewConfig{
					Border: "#EEEEDD",
					InfoPane: InfoPaneConfig{
						DeletedFrom: ColorConfig{
							Foreground: "#EEEEEE",
							Background: "#1C1C1C",
						},
						DeletedAt: ColorConfig{
							Foreground: "#EEEEEE",
							Background: "#1C1C1C",
						},
					},
					PreviewPane: PreviewPaneConfig{
						Border: "#3C3C3C",
						Size: ColorConfig{
							Foreground: "#EEEEDD",
							Background: "#3C3C3C",
						},
						Scroll: ColorConfig{
							Foreground: "#EEEEDD",
							Background: "#3C3C3C",
						},
					},
				},
				DeletionDialog: "#FF007F",
			},
		},
		History: History{
			Include: IncludeConfig{
				Period: 365,
			},
			Exclude: ExcludeConfig{
				Files: []string{
					".DS_Store",
				},
				Patterns: []string{},
				Globs:    []string{},
				Size: SizeConfig{
					Min: "0KB",
					Max: "10GB",
				},
			},
		},
	}
}
