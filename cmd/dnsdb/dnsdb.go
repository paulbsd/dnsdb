package main

import (
	"log"

	"git.paulbsd.com/paulbsd/dnsdb/src/config"
	"git.paulbsd.com/paulbsd/dnsdb/src/core"
)

const DB string = "db"
const BASEDIR string = "/etc/dnsdist/db"

func main() {
	var err error
	var configfile = config.ParseArgs()
	var dbs = config.GetCfg(configfile)
	for _, db := range dbs {
		switch db.Type {
		case "domain":
			err = core.HandleDomains(db.URL, db.File)
		case "ip":
			err = core.HandleIPs(DB, db.URL, db.File)
		}
		if err != nil {
			log.Println(err)
		}
	}
}
