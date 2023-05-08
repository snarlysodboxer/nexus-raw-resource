package commands

import (
	"io"

	"github.com/sirupsen/logrus"
)

func setupLogging(stderr io.Writer) {
	logrus.SetOutput(stderr)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
}
