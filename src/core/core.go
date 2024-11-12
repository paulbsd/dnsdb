package core

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/netip"
	"strings"

	"github.com/3th1nk/cidr"
	"github.com/colinmarc/cdb"
	"github.com/rs/zerolog"
	"wellquite.org/golmdb"
)

func GetBody(url string) (body io.ReadCloser, err error) {
	res, err := http.Get(url)
	if err != nil {
		return
	}
	if res.StatusCode != 200 {
		err = fmt.Errorf("error with %s url with http code %d", url, res.StatusCode)
		return
	}
	return res.Body, nil
}

func HandleDomains(url string, file string) (err error) {
	var handled int

	body, err := GetBody(url)
	if err != nil {
		log.Println(err)
		return
	}

	fileScanner := bufio.NewScanner(body)
	fileScanner.Split(bufio.ScanLines)

	writer, err := cdb.Create(file)
	if err != nil {
		log.Fatal(err)
	}

	for fileScanner.Scan() {
		var dom = fileScanner.Text()
		writer.Put([]byte(dom), []byte(""))
		handled++
	}
	log.Printf("%d domains handled for url %s\n", handled, url)
	writer.Close()
	return
}

func HandleIPs(db string, url string, file string) (err error) {
	body, err := GetBody(url)
	if err != nil {
		log.Println(err)
		return
	}

	fileScanner := bufio.NewScanner(body)
	fileScanner.Split(bufio.ScanLines)

	logger := zerolog.New(nil)
	client, err := golmdb.NewLMDB(logger, file, 0666, 100, 4, golmdb.NoReadAhead|golmdb.NoSubDir, 1000)
	if err != nil {
		return
	}

	err = client.Update(func(txn *golmdb.ReadWriteTxn) (err error) {
		dbRef, err := txn.DBRef(db, golmdb.Create)
		err = txn.Drop(dbRef)
		if err != nil {
			log.Println(err)
			return
		}
		return
	})
	if err != nil {
		log.Println(err)
	}

	err = client.Update(func(txn *golmdb.ReadWriteTxn) (err error) {
		var handled int
		dbRef, err := txn.DBRef(db, golmdb.Create)
		if err != nil {
			log.Println(err)
		}

		for fileScanner.Scan() {
			var upper, lower []byte
			var line = fileScanner.Text()
			var ipitem = strings.Split(line, "#")[0]
			ipitem = strings.TrimSpace(ipitem)
			if len(ipitem) == 0 {
				continue
			}
			if strings.Contains(ipitem, "/") {
				upper, lower, err = convertCIDR(ipitem)
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
			err = txn.Put(dbRef, upper, lower, 0)
			if err != nil {
				log.Println(err)
				return
			}
			handled++
		}
		log.Printf("%d ips handled for url %s\n", handled, url)
		return
	})
	return
}

func convertIP(ip string) (res []byte, err error) {
	pa, err := netip.ParseAddr(ip)
	if err != nil {
		return nil, err
	}
	res = pa.AsSlice()
	return res, err
}

func convertCIDR(iprange string) (upperres []byte, lowerres []byte, err error) {
	cp, err := cidr.Parse(iprange)
	if err != nil {
		return nil, nil, err
	}
	upper, _ := netip.AddrFromSlice(cp.Broadcast())
	lower, _ := netip.AddrFromSlice(cp.Network())
	return upper.AsSlice(), lower.AsSlice(), nil
}
