package main

import (
	"os"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"github.com/snarlysodboxer/nexus-raw-resource/commands"
)

func main() {
	color.NoColor = false

	command := commands.NewOut(os.Stdin, os.Stderr, os.Stdout, os.Args)
	err := command.Execute()
	if err != nil {
		logrus.Errorf("%s", err)
		os.Exit(1)
	}
}
