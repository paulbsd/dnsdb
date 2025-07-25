package core

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"git.paulbsd.com/paulbsd/dnsdb/src/config"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/colinmarc/cdb"
)

const lastmodifiedFormat = `Mon, 2 Jan 2006 15:04:05 MST`

func GetBody(url string) (body io.ReadCloser, lastmodified time.Time, err error) {
	lastmodified = time.Now()
	err = fmt.Errorf("Can't access data")

	if strings.HasPrefix(url, "file:///") {
		path := strings.Replace(url, "file://", "", 1)
		body, lastmodified, err = GetLocalFile(path)

		return body, lastmodified, err
	} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		body, lastmodified, err = GetRemoteFile(url)

		return body, lastmodified, nil
	}

	return
}

func GetLocalFile(path string) (body io.ReadCloser, lastmodified time.Time, err error) {
	file, err := os.Open(path)
	if err != nil {
		log.Println(err)
	}

	fstat, err := file.Stat()
	if err != nil {
		log.Println(err)
	}

	lastmodified = fstat.ModTime()
	body = file

	return
}

func GetRemoteFile(url string) (body io.ReadCloser, lastmodified time.Time, err error) {
	res, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}

	lmstr := res.Header.Get("Last-Modified")
	if err != nil {
		log.Println(err)
	}

	lastmodified, err = time.Parse(lastmodifiedFormat, lmstr)
	if err != nil {
		return nil, lastmodified, err
	}

	if res.StatusCode != 200 {
		err = fmt.Errorf("error with %s url with http code %d", url, res.StatusCode)
		return nil, lastmodified, err
	}

	body = res.Body

	return
}

func HandleStringOrDomain(cfg *config.Cfg, blocklist *config.Blocklist) (err error) {
	var handled int
	var tmpfile = fmt.Sprintf("%s.tmp", blocklist.File)

	body, lastmodified, err := GetBody(blocklist.URL)
	if err != nil {
		log.Println(err)
		return
	}

	if CompareMtimes(blocklist.File, lastmodified) {
		fileScanner := bufio.NewScanner(body)
		fileScanner.Split(bufio.ScanLines)

		{
			writer, err := cdb.Create(tmpfile)
			if err != nil {
				log.Println(err)
				log.Fatalf("can't open file %s\n", tmpfile)
			}

			defer writer.Close()

			for fileScanner.Scan() {
				var line = fileScanner.Text()
				var s = strings.TrimSpace(strings.Split(line, "#")[0])
				if len(s) > 0 {
					writer.Put([]byte(s), []byte(blocklist.DefaultValue))
					handled++
				}
			}

			log.Printf("%d domains/strings handled for url %s\n", handled, blocklist.URL)
		}

		err = os.Rename(tmpfile, blocklist.File)
		if err != nil {
			log.Fatalf("can't move file %s to %s\n", tmpfile, blocklist.File)
		}
	} else {
		log.Printf("not modifying file %s\n", blocklist.File)
	}

	return
}

func HandleIP(cfg *config.Cfg, dbname string, blocklist *config.Blocklist) (err error) {
	body, lastmodified, err := GetBody(blocklist.URL)
	if err != nil {
		log.Fatalln(err)
		return
	}

	if CompareMtimes(blocklist.File, lastmodified) {
		fileScanner := bufio.NewScanner(body)
		fileScanner.Split(bufio.ScanLines)

		env, err := lmdb.NewEnv()
		if err != nil {
			log.Println(err)
		}

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
			if err != nil {
				log.Println(err)
			}

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
	} else {
		log.Printf("not modifying file %s\n", blocklist.File)
	}

	return
}
