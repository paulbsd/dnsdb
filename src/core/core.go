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
		if err != nil {
			return
		}

		return body, lastmodified, err
	} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		body, lastmodified, err = GetRemoteFile(url)
		if err != nil {
			return
		}

		return body, lastmodified, nil
	}

	return
}

func GetLocalFile(path string) (body io.ReadCloser, lastmodified time.Time, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}

	fstat, err := file.Stat()
	if err != nil {
		return
	}

	lastmodified = fstat.ModTime()
	body = file

	return
}

func GetRemoteFile(url string) (body io.ReadCloser, lastmodified time.Time, err error) {
	client := &http.Client{}
	res, err := client.Get(url)
	if err != nil {
		log.Println(err)
	}

	lmstr := res.Header.Get("Last-Modified")
	if lmstr == "" {
		err = fmt.Errorf("no 'Last-Modified' header found!")
		return
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

func HandleStringOrDomain(cfg *config.Cfg, database *config.Database) (err error) {
	var handled int
	var tmpfile = fmt.Sprintf("%s.tmp", database.File)

	body, lastmodified, err := GetBody(database.URL)
	if err != nil {
		log.Println(err)
		return
	}

	old, err := CompareMtimes(database.File, lastmodified)
	if err != nil {
		return
	}

	if old {
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
					writer.Put([]byte(s), []byte(database.DefaultValue))
					handled++
				}
			}

			log.Printf("%d domains/strings handled for url %s\n", handled, database.URL)
		}

		err = os.Rename(tmpfile, database.File)
		if err != nil {
			log.Fatalf("can't move file %s to %s\n", tmpfile, database.File)
		}
	} else {
		log.Printf("not modifying file %s\n", database.File)
	}

	return
}

func HandleIP(cfg *config.Cfg, dbname string, database *config.Database) (err error) {
	body, lastmodified, err := GetBody(database.URL)
	if err != nil {
		return
	}
	old, err := CompareMtimes(database.File, lastmodified)
	if err != nil {
		return
	}

	if old {
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

		err = env.Open(database.File, lmdb.NoReadahead|lmdb.NoSubdir, 0664)
		if err != nil {
			log.Fatalf("can't open file %s\n", database.File)
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

			log.Printf("%d ips handled for url %s\n", handled, database.URL)

			return
		})
	} else {
		log.Printf("not modifying file %s\n", database.File)
	}

	return
}
