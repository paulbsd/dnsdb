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

func GetCfg(configfile string) (cfg Cfg, err error) {
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

type Blocklist struct {
	URL          string `yaml:"url"`
	File         string `yaml:"file"`
	Type         string `yaml:"type"`
	DefaultValue string `yaml:"default_value"`
}

type CfgItems struct {
	IPv4MaxCidrValue int         `yaml:"ipv4_max_cidr_value"`
	IPv6MaxCidrValue int         `yaml:"ipv6_max_cidr_value"`
	Blocklists       []Blocklist `yaml:"blocklists"`
}

type Cfg struct {
	Config CfgItems
}
