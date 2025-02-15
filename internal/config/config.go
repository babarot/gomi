package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/babarot/gomi/internal/env"
	"github.com/babarot/gomi/internal/utils/shell"
	"github.com/go-playground/validator/v10"
	"github.com/muesli/reflow/indent"
	"gopkg.in/yaml.v2"
)

var validate *validator.Validate

type Config struct {
	Core    Core    `yaml:"core"`
	UI      UI      `yaml:"ui"`
	History History `yaml:"history"`
}

type Core struct {
	Trash Trash `yaml:"trash"`

	// TrashDir specifies the trash directory location
	TrashDir string `yaml:"trash_dir" validate:"dirpath|allowEmpty"`

	// HomeFallback enables fallback to home trash when external trash fails
	HomeFallback bool `yaml:"home_fallback"`

	// Restore contains restore-specific settings
	Restore RestoreConfig `yaml:"restore"`

	// Verbose enables detailed output
	Verbose bool `yaml:"verbose"`
}

type Trash struct {
	Strategy string `yaml:"strategy" validate:"validStrategy|allowEmpty"`
}

type RestoreConfig struct {
	// Confirm asks for confirmation before restoring
	Confirm bool `yaml:"confirm"`

	// Verbose enables detailed output during restore
	Verbose bool `yaml:"verbose"`
}

type UI struct {
	Density     string        `yaml:"density" validate:"required,oneof=compact spacious"`
	Style       styleConfig   `yaml:"style"`
	ExitMessage string        `yaml:"exit_message"`
	Preview     previewConfig `yaml:"preview"`
	Paginator   string        `yaml:"paginator_type" validate:"required,oneof=dots arabic"`
}

type History struct {
	Include includeConfig `yaml:"include"`
	Exclude excludeConfig `yaml:"exclude"`
}

type includeConfig struct {
	Period int `yaml:"within_days"`
}

type excludeConfig struct {
	Files    []string `yaml:"files"`
	Patterns []string `yaml:"patterns"`
	Globs    []string `yaml:"globs"`
	Size     size     `yaml:"size"`
}

type size struct {
	Min string `yaml:"min" validate:"validSize|allowEmpty"`
	Max string `yaml:"max" validate:"validSize|allowEmpty"`
}

type previewConfig struct {
	SyntaxHighlight  bool   `yaml:"syntax_highlight"`
	Colorscheme      string `yaml:"colorscheme"`
	DirectoryCommand string `yaml:"directory_command"`
}

type styleConfig struct {
	ListView   ListView   `yaml:"list_view"`
	DetailView DetailView `yaml:"detail_view"`
}

type ListView struct {
	IndentOnSelect bool   `yaml:"indent_on_select"`
	Cursor         string `yaml:"cursor"`
	Selected       string `yaml:"selected"`
}

type DetailView struct {
	Border      string      `yaml:"border"`
	InfoPane    infoPane    `yaml:"info_pane"`
	PreviewPane previewPane `yaml:"preview_pane"`
}

type infoPane struct {
	DeletedFrom color `yaml:"deleted_from"`
	DeletedAt   color `yaml:"deleted_at"`
}

type previewPane struct {
	Border string `yaml:"border"`
	Size   color  `yaml:"size"`
	Scroll color  `yaml:"scroll"`
}

type color struct {
	Foreground string `yaml:"fg"`
	Background string `yaml:"bg"`
}

// Parser handles configuration file parsing and validation
type parser struct{}

func (p parser) getDefaultConfig() Config {
	return Config{
		Core: Core{
			Trash: Trash{
				Strategy: "auto",
			},
			// Default to $XDG_DATA_HOME/Trash for XDG spec
			TrashDir:     filepath.Join(os.Getenv("HOME"), ".local", "share", "Trash"),
			HomeFallback: true,
			Restore: RestoreConfig{
				Confirm: true,
				Verbose: true,
			},
			Verbose: false,
		},
		UI: UI{
			Density:     "spacious",
			ExitMessage: "bye!",
			Preview: previewConfig{
				SyntaxHighlight:  true,
				Colorscheme:      "nord",
				DirectoryCommand: "ls -GF -1 -A --color=always",
			},
			Paginator: "dots",
			Style: styleConfig{
				ListView: ListView{
					IndentOnSelect: true,
					Cursor:         "#AD58B4",
					Selected:       "#5FB458",
				},
				DetailView: DetailView{
					Border: "#EEEEDD",
					InfoPane: infoPane{
						DeletedFrom: color{
							Foreground: "#EEEEEE",
							Background: "#1C1C1C",
						},
						DeletedAt: color{
							Foreground: "#EEEEEE",
							Background: "#1C1C1C",
						},
					},
					PreviewPane: previewPane{
						Border: "#3C3C3C",
						Size: color{
							Foreground: "#EEEEDD",
							Background: "#3C3C3C",
						},
						Scroll: color{
							Foreground: "#EEEEDD",
							Background: "#3C3C3C",
						},
					},
				},
			},
		},
		History: History{
			Include: includeConfig{
				Period: 365,
			},
			Exclude: excludeConfig{
				Files: []string{
					".DS_Store",
				},
				Patterns: []string{},
				Globs:    []string{},
				Size: size{
					Min: "0KB",
					Max: "10GB",
				},
			},
		},
	}
}

type configError struct {
	configPath string
	configDir  string
	parser     parser
	err        error
}

func validStrategy(fl validator.FieldLevel) bool {
	value := strings.ToLower(fl.Field().String())
	switch value {
	case "auto", "xdg", "legacy":
		return true
	default:
		return false
	}
}

func validSize(fl validator.FieldLevel) bool {
	value := strings.ToUpper(fl.Field().String())
	re := regexp.MustCompile(`^\d+(KB|MB|GB|TB|PB)$`)
	return re.MatchString(value)
}

func allowEmpty(fl validator.FieldLevel) bool {
	str := fl.Field().String()
	return strings.TrimSpace(str) == "" || fl.Parent().FieldByName(fl.StructFieldName()).IsValid()
}

func (p parser) getDefaultConfigContents() string {
	defaultConfig := p.getDefaultConfig()
	content, _ := yaml.Marshal(defaultConfig)
	return string(content)
}

func (e configError) Error() string {
	return heredoc.Docf(`
		Couldn't find the "%s" config file.
		Please try again after creating it or specifying a valid config path.
		The recommended config path is %s (default).
		Example YAML file contents:
		---
		%s
		---
		Original error:
		%s
		`,
		e.configPath,
		env.GOMI_CONFIG_PATH,
		e.parser.getDefaultConfigContents(),
		indent.String(e.err.Error(), 2),
	)
}

func (p parser) createConfigFile(path string) error {
	// Ensure directory exists
	if err := p.ensureDirExists(filepath.Dir(path)); err != nil {
		return err
	}

	// Create the config file if missing
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slog.Warn("creating config file as it does not exist", "config-file", path)
		newConfigFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			return err
		}
		defer newConfigFile.Close()

		// Write default config contents
		if err := p.writeConfigFileContents(newConfigFile); err != nil {
			return err
		}
	}

	return nil
}

func (p parser) ensureDirExists(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		slog.Warn("creating directory as it does not exist", "dir", dirPath)
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func (p parser) writeConfigFileContents(file *os.File) error {
	_, err := file.WriteString(p.getDefaultConfigContents())
	return err
}

func (p parser) ensureConfigFile() (string, error) {
	path := env.GOMI_CONFIG_PATH

	// Ensure directory exists before creating file
	if err := p.ensureDirExists(filepath.Dir(path)); err != nil {
		return "", err
	}

	// Create file if missing
	if err := p.createConfigFile(path); err != nil {
		return "", configError{
			parser:    p,
			configDir: filepath.Dir(path),
			err:       err,
		}
	}

	return path, nil
}

type parsingError struct {
	err error
}

func (e parsingError) Error() string {
	return fmt.Sprintf("failed to parse config: %v", e.err)
}

func (p parser) readConfigFile(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, configError{
			configPath: path,
			configDir:  filepath.Dir(path),
			parser:     p,
			err:        err,
		}
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if err := validate.Struct(cfg); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			return cfg, fmt.Errorf("validation error: Field %s, %q is invalid\n", err.Field(), err.Value())
		}
	}
	return cfg, nil
}

func initParser() parser {
	validate = validator.New()
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.Split(fld.Tag.Get("yaml"), ",")[0]
		if name == "-" {
			return ""
		}
		return name
	})

	_ = validate.RegisterValidation("validStrategy", validStrategy)
	_ = validate.RegisterValidation("validSize", validSize)
	_ = validate.RegisterValidation("allowEmpty", allowEmpty)

	return parser{}
}

func Parse(path string) (Config, error) {
	parser := initParser()

	var cfg Config
	var err error
	var configPath string

	if path == "" {
		configPath, err = parser.ensureConfigFile()
		if err != nil {
			return cfg, parsingError{err: err}
		}
	} else {
		configPath = path
	}
	slog.Debug("config file found", "config-file", configPath)

	cfg, err = parser.readConfigFile(configPath)
	if err != nil {
		return cfg, parsingError{err: err}
	}

	// If using legacy format, adjust trash dir
	if cfg.Core.Trash.Strategy == "legacy" && cfg.Core.TrashDir == "" {
		cfg.Core.TrashDir = filepath.Join(os.Getenv("HOME"), ".gomi")
	}

	// expand tilda etc
	trashDir, err := shell.ExpandHome(cfg.Core.TrashDir)
	if err != nil {
		return cfg, parsingError{err: err}
	}
	cfg.Core.TrashDir = trashDir

	return cfg, nil
}
