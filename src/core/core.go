package core

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"git.paulbsd.com/paulbsd/dnsdb/src/config"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/colinmarc/cdb"
)

func GetBody(url string) (body io.ReadCloser, err error) {
	if strings.HasPrefix(url, "file:///") {
		path := strings.Replace(url, "file://", "", 1)
		body, err = os.Open(path)
		return
	} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		res, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != 200 {
			err = fmt.Errorf("error with %s url with http code %d", url, res.StatusCode)
			return nil, err
		}
		return res.Body, nil
	}

	return nil, fmt.Errorf("Can't access data")
}

func HandleStringOrDomain(cfg *config.Cfg, blocklist *config.Blocklist) (err error) {
	var handled int
	var tmpfile = fmt.Sprintf("%s.tmp", blocklist.File)

	body, err := GetBody(blocklist.URL)
	if err != nil {
		log.Println(err)
		return
	}

	fileScanner := bufio.NewScanner(body)
	fileScanner.Split(bufio.ScanLines)

	writer, err := cdb.Create(tmpfile)
	if err != nil {
		log.Fatalf("can't open file %s\n", tmpfile)
	}

	for fileScanner.Scan() {
		var line = fileScanner.Text()
		var s = strings.TrimSpace(strings.Split(line, "#")[0])
		if len(s) > 0 {
			writer.Put([]byte(s), []byte(blocklist.DefaultValue))
			handled++
		}
	}
	log.Printf("%d domains/strings handled for url %s\n", handled, blocklist.URL)
	writer.Close()

	err = os.Rename(tmpfile, blocklist.File)
	if err != nil {
		log.Fatalf("can't move file %s to %s\n", tmpfile, blocklist.File)
	}

	return
}

func HandleIP(cfg *config.Cfg, dbname string, blocklist *config.Blocklist) (err error) {
	body, err := GetBody(blocklist.URL)
	if err != nil {
		log.Fatalln(err)
		return
	}

	fileScanner := bufio.NewScanner(body)
	fileScanner.Split(bufio.ScanLines)

	env, err := lmdb.NewEnv()
	err = env.SetMapSize(100 * 1024 * 1024)
	if err != nil {
		log.Println(err)
	}
	err = env.SetMaxDBs(1)
	if err != nil {
		log.Println(err)
	}

	err = env.Open(blocklist.File, lmdb.NoReadahead|lmdb.NoSubdir, 0664)
	if err != nil {
		log.Fatalf("can't open file %s\n", blocklist.File)
	}
	defer env.Close()

	err = env.Update(func(txn *lmdb.Txn) (err error) {
		dbi, err := txn.CreateDBI(dbname)
		err = txn.Drop(dbi, false)
		if err != nil {
			log.Println(err)
			return
		}
		return
	})
	if err != nil {
		log.Println(err)
	}

	err = env.Update(func(txn *lmdb.Txn) (err error) {
		var handled int
		dbi, err := txn.CreateDBI(dbname)
		if err != nil {
			log.Println(err)
		}

		for fileScanner.Scan() {
			var upper, lower []byte
			var line = fileScanner.Text()
			var ipitem = strings.TrimSpace(strings.Split(line, "#")[0])
			if len(ipitem) == 0 {
				continue
			}
			if strings.Contains(ipitem, "/") {
				upper, lower, err = convertCIDR(ipitem, cfg.Config.IPv4MaxCidrValue, cfg.Config.IPv6MaxCidrValue)
				if err != nil {
					log.Println(err)
					continue
				}
			} else {
				upper, err = convertIP(ipitem)
				if err != nil {
					log.Println(err)
					continue
				}
				lower = upper
			}
			err = txn.Put(dbi, upper, lower, 0)
			if err != nil {
				log.Println(err)
				return
			}
			handled++
		}
		log.Printf("%d ips handled for url %s\n", handled, blocklist.URL)
		return
	})

	return
}
