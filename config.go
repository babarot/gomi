package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-playground/validator"
	"gopkg.in/yaml.v2"
)

const gomiConfigDir = "gomi"
const gomiConfigFilename = "config.yaml"
const DEFAULT_XDG_CONFIG_DIRNAME = ".config"

var validate *validator.Validate

type Config struct {
	ShowDescription bool     `yaml:"show_description"`
	ExcludeItems    []string `yaml:"exclude_items"`
}

type configError struct {
	configDir string
	parser    ConfigParser
	err       error
}

type ConfigParser struct{}

func (parser ConfigParser) getDefaultConfig() Config {
	return Config{
		ShowDescription: true,
		ExcludeItems: []string{
			// In macOS, .DS_Store is a file that stores custom attributes of its
			// containing folder, such as folder view options, icon positions,
			// and other visual information
			".DS_Store",
		},
	}
}

func (parser ConfigParser) getDefaultConfigYamlContents() string {
	defaultConfig := parser.getDefaultConfig()
	content, _ := yaml.Marshal(defaultConfig)
	return string(content)
}

func (e configError) Error() string {
	return fmt.Sprintf(
		`Couldn't find a config.yml or a config.yaml configuration file.
Create one under: %s

Example of a config.yml file:
%s

Original error: %v`,
		path.Join(e.configDir, gomiConfigDir, gomiConfigFilename),
		string(e.parser.getDefaultConfigYamlContents()),
		e.err,
	)
}

func (parser ConfigParser) writeDefaultConfigContents(
	newConfigFile *os.File,
) error {
	_, err := newConfigFile.WriteString(parser.getDefaultConfigYamlContents())

	if err != nil {
		return err
	}

	return nil
}

func (parser ConfigParser) createConfigFileIfMissing(
	configFilePath string,
) error {
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		newConfigFile, err := os.OpenFile(
			configFilePath,
			os.O_RDWR|os.O_CREATE|os.O_EXCL,
			0666,
		)
		if err != nil {
			return err
		}

		defer newConfigFile.Close()
		return parser.writeDefaultConfigContents(newConfigFile)
	}

	return nil
}

func (parser ConfigParser) getDefaultConfigFileOrCreateIfMissing() (string, error) {
	var configFilePath string
	gomiConfigPath := os.Getenv("GOMI_CONFIG_PATH")

	// First try GOMI_CONFIG_PATH
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
				parser:    parser,
				configDir: configDir,
				err:       err,
			}
		}
	}

	if err := parser.createConfigFileIfMissing(configFilePath); err != nil {
		return "", configError{parser: parser, configDir: configDir, err: err}
	}

	return configFilePath, nil
}

type parsingError struct {
	err error
}

func (e parsingError) Error() string {
	return fmt.Sprintf("failed parsing config.yml: %v", e.err)
}

func (parser ConfigParser) readConfigFile(path string) (Config, error) {
	config := parser.getDefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return config, configError{parser: parser, configDir: path, err: err}
	}

	err = yaml.Unmarshal([]byte(data), &config)
	if err != nil {
		return config, err
	}

	err = validate.Struct(config)
	return config, err
}

func initParser() ConfigParser {
	validate = validator.New()

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.Split(fld.Tag.Get("yaml"), ",")[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return ConfigParser{}
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
