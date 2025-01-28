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
	Restore Restore `yaml:"restore"`
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

type Restore struct {
	Verbose bool `yaml:"verbose"`
	Confirm bool `yaml:"confirm"`
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
	Min string `yaml:"min" validate:"validSize"`
	Max string `yaml:"max" validate:"validSize"`
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

type configError struct {
	configPath string
	configDir  string
	parser     parser
	err        error
}

type parser struct{}

func validSize(fl validator.FieldLevel) bool {
	value := strings.ToUpper(fl.Field().String())
	re := regexp.MustCompile(`^\d+(KB|MB|GB|TB|PB)|$`) // empty is acceptable
	return re.MatchString(value)
}

func (p parser) getDefaultConfig() Config {
	return Config{
		Core: Core{
			Restore: Restore{
				Verbose: true,
				Confirm: true,
			},
		},
		UI: UI{
			Density:     "spacious", // or compact
			ExitMessage: "bye!",
			Preview: previewConfig{
				SyntaxHighlight:  true,
				Colorscheme:      "nord",
				DirectoryCommand: "ls -GF -1 -A --color=always",
			},
			Paginator: "dots", // or arabic
			Style: styleConfig{
				ListView: ListView{
					IndentOnSelect: true,
					Cursor:         "#AD58B4", // Purple
					Selected:       "#5FB458", // Green
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
					// In macOS, .DS_Store is a file that stores custom attributes of its
					// containing folder, such as folder view options, icon positions,
					// and other visual information
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

	_ = validate.RegisterValidation("validSize", validSize)

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

	return cfg, nil
}
