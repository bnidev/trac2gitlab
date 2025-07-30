package main

import (
	"trac2gitlab/internal/cli"
	"trac2gitlab/internal/utils"
)

func main() {
	utils.InitLogger()
	cli.Execute()
}
