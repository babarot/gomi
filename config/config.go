package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/babarot/gomi/env"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
)

var validate *validator.Validate

type Config struct {
	UI        UI        `yaml:"ui"`
	Inventory Inventory `yaml:"inventory"`
	Restore   Restore   `yaml:"restore"`
}

type UI struct {
	Density    string        `yaml:"density"`
	Style      styleConfig   `yaml:"style"`
	ByeMessage string        `yaml:"bye_message"`
	Preview    previewConfig `yaml:"preview"`
	Paginator  string        `yaml:"paginator_style"`
}

type Inventory struct {
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
	Files     []string `yaml:"files"`
	Patterns  []string `yaml:"patterns"`
	Globs     []string `yaml:"globs"`
	SizeAbove []string `yaml:"size_above"` // over
	SizeBelow []string `yaml:"size_below"` // under
}

type previewConfig struct {
	SyntaxHighlight  bool   `yaml:"syntax_highlight"`
	Colorscheme      string `yaml:"colorscheme"`
	DirectoryCommand string `yaml:"directory_command"`
}

type styleConfig struct {
	Window      window      `yaml:"window"`
	PreviewPane previewPane `yaml:"preview_pane"`
}
type window struct {
	Border  string `yaml:"border"`
	Section color  `yaml:"section"`
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
	configDir string
	parser    parser
	err       error
}

type parser struct{}

func (p parser) getDefaultConfig() Config {
	return Config{
		UI: UI{
			Density:    "compact | spacious",
			ByeMessage: "bye!",
			Preview: previewConfig{
				SyntaxHighlight:  true,
				Colorscheme:      "nord",
				DirectoryCommand: "ls -GF -1 -A --color=always",
			},
			Paginator: "dots | arabic",
			Style: styleConfig{
				Window: window{
					Border: "#EEEEDD",
					Section: color{
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
		Inventory: Inventory{
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
				Patterns:  []string{},
				Globs:     []string{},
				SizeAbove: []string{"10GB"},
				SizeBelow: []string{"0KB"},
			},
		},
		Restore: Restore{
			Verbose: true,
			Confirm: true,
		},
	}
}

func (p parser) getDefaultConfigYamlContents() string {
	defaultConfig := p.getDefaultConfig()
	content, _ := yaml.Marshal(defaultConfig)
	return string(content)
}

func (e configError) Error() string {
	return heredoc.Docf(`
		Couldn't find a "config.yaml" config file.
		Create one under: %s
		Example of a config.yaml file:

		%s

		The detail error is: %v`,
		"path.Join(e.configDir, gomiConfigDir, gomiConfigFilename)",
		string(e.parser.getDefaultConfigYamlContents()),
		e.err,
	)
}

func (p parser) writeDefaultConfigContents(newConfigFile *os.File) error {
	_, err := newConfigFile.WriteString(p.getDefaultConfigYamlContents())
	if err != nil {
		return err
	}
	return nil
}

func (p parser) createConfigFileIfMissing(configFilePath string) error {
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		slog.Warn(fmt.Sprintf("config file %s does not exist. creating...", configFilePath))
		newConfigFile, err := os.OpenFile(configFilePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			return err
		}
		defer newConfigFile.Close()
		return p.writeDefaultConfigContents(newConfigFile)
	}

	return nil
}

func (p parser) getDefaultConfigFileOrCreateIfMissing() (string, error) {
	path := env.GOMI_CONFIG_PATH

	// Ensure directory exists before attempting to create file
	configDir := filepath.Dir(path)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		slog.Warn(fmt.Sprintf("configDir %s does not exist. creating...", configDir))
		if err = os.MkdirAll(configDir, os.ModePerm); err != nil {
			return "", configError{
				parser:    p,
				configDir: configDir,
				err:       err,
			}
		}
	}

	if err := p.createConfigFileIfMissing(path); err != nil {
		return "", configError{
			parser:    p,
			configDir: configDir,
			err:       err,
		}
	}
	return path, nil
}

type parsingError struct {
	err error
}

func (e parsingError) Error() string {
	return fmt.Sprintf("failed parsing config.yaml: %v", e.err)
}

func (p parser) readConfigFile(path string) (Config, error) {
	config := p.getDefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return config, configError{parser: p, configDir: path, err: err}
	}

	err = yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		return config, err
	}

	err = validate.Struct(config)
	return config, err
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

	return parser{}
}

func Parse(path string) (Config, error) {
	parser := initParser()

	var config Config
	var err error
	var configFilePath string

	if path == "" {
		configFilePath, err = parser.getDefaultConfigFileOrCreateIfMissing()
		if err != nil {
			return config, parsingError{err: err}
		}
	} else {
		configFilePath = path
	}
	slog.Debug(fmt.Sprintf("config found: %s. parsing", configFilePath))

	config, err = parser.readConfigFile(configFilePath)
	if err != nil {
		return config, parsingError{err: err}
	}

	return config, nil
}
