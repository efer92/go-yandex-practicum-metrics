package main

import (
	"log"
	"os"
)

func main() {
	// allowed in main of package main
	if false {
		log.Fatal("never")
		os.Exit(0)
	}
}

func helper() {
	log.Fatal("nope") // want "log.Fatal is forbidden outside main\\(\\) of package main"
	os.Exit(1)       // want "os.Exit is forbidden outside main\\(\\) of package main"
}
