package gomi

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
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

func readYaml() (c Config, err error) {
	if _, err = os.Stat(rm_config); err != nil {
		err = ioutil.WriteFile(rm_config, []byte(config_raw), os.ModePerm)
		if err != nil {
			return
		}
	}

	buf, err := ioutil.ReadFile(rm_config)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(buf, &c)
	if err != nil {
		str := []byte(err.Error())
		assigned := regexp.MustCompile(`(line \d+)`)
		group := assigned.FindSubmatch(str)
		if len(group) != 0 {
			err = fmt.Errorf("Syntax Error at %s in %s", string(group[0]), rm_config)
		}
		return
	}

	return
}
