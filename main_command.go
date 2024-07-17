package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"mydocker/cgroups/subsystems"
	"mydocker/container"
)

// 这里定义了runCommand的Flags，其作用类似于运行命令时使用--来指定参数
var runCommand = cli.Command{
	Name: "run",
	Usage: `create a container with namespace and cgroups
					limit mydocker run -ti [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti", //开启终端交互
			Usage: "enable tty",
		},
		cli.BoolFlag{
			Name:  "d", //后台执行
			Usage: "detach container",
		},
		cli.StringFlag{ //限制内存
			Name:  "m",
			Usage: "memory limit",
		},

		cli.StringFlag{ //挂存储
			Name:  "v",
			Usage: "volume",
		},

		//提供run后面的-name指定容器名字参数
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
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
		//cmd := context.Args().Get(0)
		var cmdArray []string
		for _, arg := range context.Args() {
			cmdArray = append(cmdArray, arg)
		}

		createTty := context.Bool("ti")
		detach := context.Bool("d")

		//这里的createTty和detach不能共存
		if createTty && detach {
			return fmt.Errorf("ti and d paramter can not both provided")
		}

		//打印comArray
		logrus.Infof("commandArray: %v", cmdArray)
		//tty := context.Bool("ti")

		resConf := &subsystems.ResourceConfig{
			MemoryLimit: context.String("m"),
		}

		//把volume参数传给Run函数
		volume := context.String("v")

		//将取到的容器名称传递辖区，如果没有则取到的值为空
		containerName := context.String("name")
		Run(createTty, cmdArray, resConf, volume, containerName)
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
		//cmd := context.Args().Get(0)
		//logrus.Infof("command %s", cmd)
		err := container.RunContainerInitProcess()
		return err
	},
}

// docker commit 保存镜像
var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "commit a container into image",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		imageName := context.Args().Get(0)
		CommitContainer(imageName)
		return nil
	},
}

// docker ps 查看容器信息
var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all the containers",
	Action: func(context *cli.Context) error {
		ListContainers()
		return nil
	},
}

// docker logs 查看容器日志
var logCommand = cli.Command{
	Name:  "logs",
	Usage: "print logs of a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		logContainer(containerName)
		return nil
	},
}
