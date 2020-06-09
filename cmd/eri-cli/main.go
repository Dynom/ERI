package main

import (
	"github.com/Dynom/ERI/cmd/eri-cli/commands"
)

var Version = "dev"

func main() {
	commands.SetVersion(Version)
	commands.Execute()
}
