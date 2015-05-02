package main

import (
	"fmt"
	"io/ioutil"
	"os"

	//"yaml"
	"github.com/b4b4r07/gomi/yaml"

	//"github.com/beego/goyaml2"
)

func nodeToMap(node interface{}) map[string]interface{} {
	m, ok := node.(map[string]interface{})
	if !ok {
		panic(fmt.Sprintf("%v is not of type map", node))
	}
	return m
}

func nodeToList(node interface{}) []interface{} {
	m, ok := node.([]interface{})
	if !ok {
		panic(fmt.Sprintf("%v is not of type list", node))
	}
	return m
}

var rm_config string = rm_trash + "/config.yaml"
var config_raw string = `ignore_files:
  - .DS_Store
`

func config() []interface{} {
	if _, err := os.Stat(rm_config); err != nil {
		ioutil.WriteFile(rm_config, []byte(config_raw), os.ModePerm)
	}

	file, err := os.Open(rm_config)
	if err != nil {
		panic(err)
	}
	object, err := yaml.Read(file)
	if err != nil {
		panic(err)
	}
	//value := nodeToMap(nodeToMap(object)["config"])["admin"]
	//value := nodeToMap(nodeToMap(object)["mapping"])["key1"]
	//value := nodeToMap(object)
	ignore_files := nodeToList(nodeToMap(object)["ignore_files"])
	//fmt.Printf("%s\n", ignore_files)
	return ignore_files
	//fmt.Printf("%v\n", Keys(value))
}

func Keys(m map[string]interface{}) (keys []string) {
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
