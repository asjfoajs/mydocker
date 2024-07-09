package main

import (
	"github.com/sirupsen/logrus"
	"mydocker/container"
	"os"
)

/*
*这里的Start方法是真正开始前面创建好的command的调用，它首先会clone出来一个namespace隔离的
进程，然后再子进程中，调用/proc/self/exe，也就是调用自己，发送init参数，调用我们写的init方法，去初始化容器的一些资源。
*/
func Run(tty bool, command string) {
	logrus.Infof("Run command %s", command)
	parent := container.NewParentProcess(tty, command)
	if err := parent.Start(); err != nil {
		logrus.Error("start parent process error %v", err)
	}

	parent.Wait()
	os.Exit(-1)
}
