package fataloutside

import (
	"log"
	"os"
)

func badFatal() {
	log.Fatal("nope")  // want "log.Fatal is forbidden outside main\\(\\) of package main"
	log.Fatalf("nope") // want "log.Fatalf is forbidden outside main\\(\\) of package main"
	log.Fatalln("no")  // want "log.Fatalln is forbidden outside main\\(\\) of package main"
}

func badExit() {
	os.Exit(1) // want "os.Exit is forbidden outside main\\(\\) of package main"
}
