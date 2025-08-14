package main

import (
	"github.com/bnidev/trac2gitlab/internal/cli"
	"github.com/bnidev/trac2gitlab/internal/utils"
)

func main() {
	utils.InitLogger()
	cli.Execute()
}
