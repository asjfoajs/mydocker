package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"mydocker/container"
)

// 这里定义了runCommand的Flags，其作用类似于运行命令时使用--来指定参数
var runCommand = cli.Command{
	Name: "run",
	Usage: `create a container with namespace and cgroups
					limit mydocker run -ti [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti",
			Usage: "enable tty",
		},
	},

	/**
	这里是run命令执行的真正函数
	1.判断参数是否包含command
	2.获取用户指定的command
	3.调用Run funcation 去准备启动容器
	*/
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("Missing container command")
		}
		cmd := context.Args().Get(0)
		tty := context.Bool("ti")
		Run(tty, cmd)
		return nil
	},
}

// 这里定义了intiCommand的具体操作，此操作为内部方法，禁止外部调用
var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container.Do not call it outside",
	/**
	1.获取传递过来的command参数
	2.执行容器初始化
	*/
	Action: func(context *cli.Context) error {
		logrus.Infof("init come on")
		cmd := context.Args().Get(0)
		logrus.Infof("command %s", cmd)
		err := container.RunContainerInitProcess(cmd, nil)
		return err
	},
}
