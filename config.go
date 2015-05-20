package gomi

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Root   string   `yaml:"root"`
	Ignore []string `yaml:"ignore_files"`
}

var rm_config string = filepath.Join(rm_trash, "config.yaml")
var config_raw string = `root: ~/.gomi

# Interpret if name matches the shell file name pattern
ignore_files:
  - .DS_Store
  - "*~"
`

func (q *Config) ReadConfig() error {
	setting := []byte(config_raw)

	if _, err := os.Stat(rm_config); err == nil {
		setting, err = ioutil.ReadFile(rm_config)
		if err != nil {
			return err
		}
	} else {
		err = ioutil.WriteFile(rm_config, []byte(config_raw), os.ModePerm)
		if err != nil {
			return err
		}
	}

	var data = &q
	err := yaml.Unmarshal(setting, data)

	if err != nil {
		str := []byte(err.Error())
		assigned := regexp.MustCompile(`(line \d+)`)
		group := assigned.FindSubmatch(str)
		if len(group) != 0 {
			err = fmt.Errorf("Syntax Error at %s in %s", string(group[0]), rm_config)
		}
		return err
	}

	return nil
}
