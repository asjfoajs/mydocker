package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

///*
//*
//这里的init函数是在容器内部执行的，也就是说，代码执行到这里后，容器所在的进程其实就已经创建出来了，
//这是本容器执行的第一个进程。
//使用mount先去挂载proc文件系统，以便后面通过ps等系统命令去查看当前进程资源的情况。
//*/
//func RunContainerInitProcess(command string, args []string) error {
//	//logrus.Infof("command %s", command)
//	logrus.Infof("RunContainerInitProcess args %s", args)
//
//	////MS_NOEXEC 在本文件系统中不允许运行其他程序
//	////MS_NOSUID 在本系统中运行程序的时候，不允许set-user-ID或set-group-ID
//	////MS_NODEV 这个参数是从linux2.4以来，所有mount的系统都会默认设定的参数。
//	//defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
//	//
//	//err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
//	//if err != nil {
//	//	logrus.Errorf("mount proc err %v", err)
//	//	return err
//	//}
//
//	argv := []string{command}
//
//	if err := syscall.Exec(command, argv, os.Environ()); err != nil {
//		logrus.Errorf(err.Error())
//	}
//	return nil
//}

func RunContainerInitProcess() error {
	//time.Sleep(5 * time.Second)
	//打印RunContainerInitProcess函数
	logrus.Infof("RunContainerInitProcess")
	cmdArray := readUserCommand()
	if cmdArray == nil || len(cmdArray) == 0 {
		return fmt.Errorf("Run container get user commadn error,cmdArray is nil")
	}
	//打印一下cmdArray
	logrus.Infof("commandArray %s", cmdArray)

	//setUpMount() 就是挂proc
	//err := syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	//	//if err != nil {
	//	//	logrus.Errorf("mount proc err %v", err)
	//	//	return err
	//	//}

	//改动，调用exec.LookPath，可以在系统的PATH里面寻找命令的绝对路径
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		logrus.Errorf("Exec loop error %v", err)
	}
	logrus.Infof("Find path %s", path)
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		logrus.Errorf(err.Error())
	}
	return nil
}

func readUserCommand() []string {
	//uintptr(3)就是指的index为3的文件描述符，也就是传递进来的管道的一端
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")

}
