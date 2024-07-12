package container

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

/*
这里是父进程，也就是当前进程执行的内容，根据上一章介绍的内容，应该比较容易明白
1.这里的/proc/self/exe调用中，/proc/self/指的是当前运行进程的自己的环境
，exec其实就是自己调用了自己，使用这种方式对创建出来的进程进行初始化
2.后面args是参数，其中init是传递给本进程的第一个参数，在本例中，其实就是会去调用initCommand去初始化
进程的一些环境和资源
3.下面的clone参数就是去fork出来一个新进程，并且使用了namespace隔离创建的进程和外部环境。
4.如果用户指定了 -ti 参数，就需要把当前进程的输入输出导入到标准输入输出上
*/
func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {
	//logrus.Infof("NewParentProcess: %s", command)
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		logrus.Errorf("New pipe error %v", err)
		return nil, nil
	}

	//args := []string{"init", command}

	//fork出来的新进程内的初始命令,默认使用sh来执行
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | //创建一个uts namespace
			syscall.CLONE_NEWIPC | //创建一个ipc namespace
			syscall.CLONE_NEWPID | //创建一个pid namespace
			syscall.CLONE_NEWNS | //创建一个mount namespace
			//syscall.CLONE_NEWUSER | //创建一个user namespace
			syscall.CLONE_NEWNET, // 创建一个network namespace
	}

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	//注意，改动在这里，在这个地方传入管道文件读取端的句柄
	//这个属性的意思是会外带着这个文件句柄去创建子进程。为什么叫“外带着” 呢?
	//因为1个进程默认会有3个文件描述符,分别是标准输入、标准输出、标准错误。这3个是子进程一
	//创建的时候就会默认带着的,那么外带的这个文件描述符理所当然地就成为了第4个。
	cmd.ExtraFiles = []*os.File{readPipe}
	return cmd, writePipe
}

// NewPipe 使用Go提供的pipe方法生成一个匿名管道。
// 这个函数返回两个变量,一个是读一个是写,其类型都是文件类型。
func NewPipe() (*os.File, *os.File, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}
