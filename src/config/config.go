package config

import (
	"flag"
	"io"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

func ParseArgs() string {
	var configfile string
	flag.StringVar(&configfile, "configfile", "dnsdb.yml", "Configuration file to use")
	flag.Parse()
	return configfile
}

func GetCfg(configfile string) (cfg []Cfg) {
	f, err := os.Open(configfile)
	if err != nil {
		log.Println(err)
		return
	}

	data, err := io.ReadAll(f)
	if err != nil {
		log.Println(err)
		return
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

type Cfg struct {
	URL  string
	File string
	Type string
}
