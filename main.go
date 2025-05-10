package main

import (
	_ "embed"
	"log"

	"github.com/sestrella/iecs/cmd"
)

//go:embed version.txt
var version string

func main() {
	err := cmd.Execute(version)
	if err != nil {
		log.Fatal(err)
	}
}
