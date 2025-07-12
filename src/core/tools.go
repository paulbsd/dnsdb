package core

import (
	"log"
	"os"
	"time"
)

func CompareMtimes(file string, lastmodified time.Time) bool {
	var currentmtime time.Time
	currentfile, err := os.Open(file)
	if err != nil {
		log.Println(err)
	} else {
		s, err := currentfile.Stat()
		if err != nil {
			log.Println(err)
		}
		currentmtime = s.ModTime()
	}
	defer currentfile.Close()

	return currentmtime.Before(lastmodified)
}
