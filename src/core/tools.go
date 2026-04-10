package core

import (
	"log"
	"os"
	"time"
)

func CompareMtimes(file string, lastmodified time.Time) (isbefore bool, err error) {
	var currentmtime time.Time
	currentfile, err := os.Open(file)
	if err != nil {
		return
	} else {
		defer currentfile.Close()
		s, err := currentfile.Stat()
		if err != nil {
			log.Println(err)
		}
		currentmtime = s.ModTime()
	}
	isbefore = currentmtime.Before(lastmodified)

	return
}
