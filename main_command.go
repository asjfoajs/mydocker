package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"mydocker/cgroups/subsystems"
	"mydocker/container"
	"os"
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

		//设置环境变量
		cli.StringSliceFlag{
			Name:  "e",
			Usage: "set container's environment",
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

		//获取镜像名
		imageName := cmdArray[0]
		cmdArray = cmdArray[1:]

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

		envSlice := context.StringSlice("e")
		Run(createTty, volume, containerName, imageName, &cmdArray, &envSlice, resConf)
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
		containerName := context.Args().Get(0)
		imageName := context.Args().Get(1)
		CommitContainer(containerName, imageName)
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

// docker exec 进入容器
var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(context *cli.Context) error {
		//非常重要如果是exec的命令并且设置了环境变量，说明是上一次exec调用的，就是为了触发c语言的那个senns
		if os.Getenv(ENV_EXEC_PID) != "" {
			logrus.Infof("pid callback pid %s", os.Getegid())
			return nil
		}

		//我们希望命令格式是mydocker exec 容器名 命令
		if len(context.Args()) < 2 {
			return fmt.Errorf("missing container name and command")
		}

		containerName := context.Args().Get(0)
		var commandArray []string
		//将除了容器名之外的参数当作需要执行的命令处理
		for _, arg := range context.Args().Tail() {
			commandArray = append(commandArray, arg)
		}
		//执行命令
		ExecContainer(containerName, commandArray)
		return nil
	},
}

// docker stop 停止容器
var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop a container",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		stopContainer(containerName)
		return nil
	},
}

// docker rm 删除容器
var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "remove one or more containers",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := context.Args().Get(0)
		removeContainer(containerName)
		return nil
	},
}
