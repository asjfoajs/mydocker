package main

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

const usage = `mydocker is a simple container runtime implementation.
					         The purpose of this project is to learn how docker works and how to write a docker by ourselves
							 Enjoy it,just for fun`

func main() {

	app := cli.NewApp()
	app.Usage = usage
	app.Name = "mydocker"

	app.Commands = []cli.Command{
		initCommand,
		runCommand,
	}

	app.Before = func(context *cli.Context) error {
		logrus.SetFormatter(&logrus.JSONFormatter{})
		logrus.SetOutput(os.Stdout)
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}

}
