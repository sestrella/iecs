package main

import (
	_ "embed"

	"github.com/sestrella/iecs/cmd"
)

//go:embed version.txt
var version string

func main() {
	err := cmd.Execute(version)
	if err != nil {
		panic(err)
	}
}
