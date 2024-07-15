package main

import (
	"github.com/sirupsen/logrus"
	"mydocker/cgroups"
	"mydocker/cgroups/subsystems"
	"mydocker/container"
	"os"
	"strings"
)

/*
*这里的Start方法是真正开始前面创建好的command的调用，它首先会clone出来一个namespace隔离的
进程，然后再子进程中，调用/proc/self/exe，也就是调用自己，发送init参数，调用我们写的init方法，去初始化容器的一些资源。
*/
//func Run(tty bool, command string) {
//	logrus.Infof("Run command %s", command)
//	parent := container.NewParentProcess(tty, command)
//	if err := parent.Start(); err != nil {
//		logrus.Error("start parent process error %v", err)
//	}
//
//	parent.Wait()
//	os.Exit(-1)
//}

func Run(tty bool, comArray []string, res *subsystems.ResourceConfig) {
	//logrus.Infof("Run command %s", command)
	parent, wirtePipe := container.NewParentProcess(tty)
	if parent == nil {
		logrus.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		logrus.Error("start parent process error %v", err)
	}

	//创建cgroup manager，并通过调用set和apply设置资源限制并使限制在容器上生效
	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup")
	defer cgroupManager.Destroy()
	//设置资源限制
	cgroupManager.Set(res)
	//将容器进程加入到各个subsystem挂载对应的cgroup中
	cgroupManager.Apply(parent.Process.Pid)
	//对容器设置完限制之后初始化容器
	sendInitCommand(comArray, wirtePipe)
	parent.Wait()

	//卸载并删除
	mntURL := "/root/mnt"
	workURL := "/root/worker"
	rootURL := "/root"

	container.DeleteWorkSpace(rootURL, mntURL, workURL)

	os.Exit(0)
}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	logrus.Infof("command all is： %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}
