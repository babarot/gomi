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
				Strategy:     "auto",
				HomeFallback: true,
				GomiDir:      filepath.Join(homedir, ".gomi"),
				ForbiddenPaths: []ForbiddenPath{
					// Default trash-related paths
					{Path: "$HOME/.local/share/Trash", Recursive: true},
					{Path: "$HOME/.trash", Recursive: true},
					{Path: "$XDG_DATA_HOME/Trash", Recursive: true},
					{Path: "/tmp/Trash", Recursive: true},
					{Path: "/var/tmp/Trash", Recursive: true},
					// gomi dir
					{Path: "$HOME/.gomi", Recursive: true},
					// Critical system directories
					{Path: "/", Recursive: false},
					{Path: "/etc", Recursive: true},
					{Path: "/usr", Recursive: true},
					{Path: "/var", Recursive: true},
					{Path: "/bin", Recursive: true},
					{Path: "/sbin", Recursive: true},
					{Path: "/lib", Recursive: true},
					{Path: "/lib64", Recursive: true},
				},
			},
			Restore: RestoreConfig{
				Confirm: true,
				Verbose: true,
			},
			PermanentDelete: PermanentDeleteConfig{
				Enable: false,
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
					FilterMatch:    "#F39C12",
					FilterPrompt:   "#7AA2F7",
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
					Max: "10GB",
				},
			},
		},
		Logging: Logging{
			Enabled: true,
			Level:   "debug",
			Rotation: RotationConfig{
				MaxSize:  "10MB",
				MaxFiles: 3,
			},
		},
	}
}
