package main

import (
	"github.com/byteowlz/ppr/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Execute(version, commit, date)
}
