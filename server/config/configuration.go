package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type Configuration struct {
	Etcd etcdConfiguration
}

type etcdConfiguration struct {
	Endpoints []string
	Timeout   int
}

func Loadconfig(path string) *Configuration {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("Config File Missing. ", err)
	}

	var config Configuration
	err = yaml.Unmarshal(file, &config)

	if err != nil {
		log.Fatal("Config Parse Error: ", err)
	}

	return &config
}
