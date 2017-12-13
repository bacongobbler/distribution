package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	logrus.SetOutput(os.Stderr)
	app := cli.NewApp()
	app.Name = "dist"
	app.Usage = "Package and ship Docker content"

	app.Action = commandList.Action
	app.Commands = []cli.Command{
		commandList,
		commandMount,
	}

	app.RunAndExitOnError()
}
