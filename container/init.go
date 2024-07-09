package container

import (
	"github.com/sirupsen/logrus"
	"os"
	"syscall"
)

/*
*
这里的init函数是在容器内部执行的，也就是说，代码执行到这里后，容器所在的进程其实就已经创建出来了，
这是本容器执行的第一个进程。
使用mount先去挂载proc文件系统，以便后面通过ps等系统命令去查看当前进程资源的情况。
*/
func RunContainerInitProcess(command string, args []string) error {
	//logrus.Infof("command %s", command)
	logrus.Infof("RunContainerInitProcess args %s", args)

	//MS_NOEXEC 在本文件系统中不允许运行其他程序
	//MS_NOSUID 在本系统中运行程序的时候，不允许set-user-ID或set-group-ID
	//MS_NODEV 这个参数是从linux2.4以来，所有mount的系统都会默认设定的参数。
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

	err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	logrus.Errorf("mount proc err %v", err)
	argv := []string{command}

	if err := syscall.Exec(command, argv, os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}
	return nil
}
