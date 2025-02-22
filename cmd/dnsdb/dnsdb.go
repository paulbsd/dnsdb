package main

import (
	"log"
	"os"

	"git.paulbsd.com/paulbsd/dnsdb/src/config"
	"git.paulbsd.com/paulbsd/dnsdb/src/core"
)

const DBNAME string = "db"
const BASEDIR string = "/etc/dnsdist/db"

func main() {
	var err error
	var configfile = config.ParseArgs()
	cfg, err := config.GetCfg(configfile)
	if err != nil {
		os.Exit(1)
	}
	for _, db := range cfg.Config.Blocklists {
		switch db.Type {
		case "domain":
			err = core.HandleStringOrDomain(&cfg, &db)
		case "string":
			err = core.HandleStringOrDomain(&cfg, &db)
		case "ip":
			err = core.HandleIP(&cfg, DBNAME, &db)
		}
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}
}
