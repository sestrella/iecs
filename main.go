package main

import (
	_ "embed"
	"log"

	"github.com/sestrella/iecs/cmd"
)

//go:embed version.txt
var version string

func main() {
	if err := cmd.Execute(version); err != nil {
		log.Fatal(err)
	}
}
