package main

import (
	"log"
	"os"

	"git.paulbsd.com/paulbsd/dnsdb/src/config"
	"git.paulbsd.com/paulbsd/dnsdb/src/core"
)

const DB string = "db"
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
			err = core.HandleDomains(&cfg, db.URL, db.File)
		case "ip":
			err = core.HandleIPs(&cfg, DB, db.URL, db.File)
		}
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}
}
