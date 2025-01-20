package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/go-playground/validator"
	"gopkg.in/yaml.v2"
)

const gomiConfigDir = "gomi"
const gomiConfigFilename = "config.yaml"
const envGomiConfigPath = "GOMI_CONFIG_PATH"

const DEFAULT_XDG_CONFIG_DIRNAME = ".config"

var validate *validator.Validate

type Config struct {
	UI        ConfigUI        `yaml:"ui"`
	Inventory ConfigInventory `yaml:"inventory"`
}

type ConfigInventory struct {
	Exclude ConfigExclude `yaml:"exclude"`
}

type ConfigExclude struct {
	Files     []string `yaml:"files"`
	Patterns  []string `yaml:"patterns"`
	Globs     []string `yaml:"globs"`
	SizeAbove []string `yaml:"size_above"` // over
	SizeBelow []string `yaml:"size_below"` // under
}

type ConfigUI struct {
	ShowDescription bool `yaml:"show_description"`
}

type configError struct {
	configDir string
	parser    parser
	err       error
}

type parser struct{}

func (p parser) getDefaultConfig() Config {
	return Config{
		UI: ConfigUI{
			ShowDescription: true,
		},
		Inventory: ConfigInventory{
			Exclude: ConfigExclude{
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
		path.Join(e.configDir, gomiConfigDir, gomiConfigFilename),
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
	var configFilePath string
	gomiConfigPath := os.Getenv(envGomiConfigPath)

	// First try env
	if gomiConfigPath != "" {
		configFilePath = gomiConfigPath
	}

	// Then fallback to global config
	if configFilePath == "" {
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			configDir = filepath.Join(homeDir, DEFAULT_XDG_CONFIG_DIRNAME)
		}

		configFilePath = filepath.Join(configDir, gomiConfigDir, gomiConfigFilename)
	}

	// Ensure directory exists before attempting to create file
	configDir := filepath.Dir(configFilePath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err = os.MkdirAll(configDir, os.ModePerm); err != nil {
			return "", configError{
				parser:    p,
				configDir: configDir,
				err:       err,
			}
		}
	}
	if err := p.createConfigFileIfMissing(configFilePath); err != nil {
		return "", configError{
			parser:    p,
			configDir: configDir,
			err:       err,
		}
	}

	return configFilePath, nil
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

func parseConfig(path string) (Config, error) {
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

	config, err = parser.readConfigFile(configFilePath)
	if err != nil {
		return config, parsingError{err: err}
	}

	return config, nil
}
