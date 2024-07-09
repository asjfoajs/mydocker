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
func NewParentProcess(tty bool, command string) *exec.Cmd {
	logrus.Infof("NewParentProcess: %s", command)

	args := []string{"init", command}
	//1.UTS Namespace 主要用来隔离 hostname（主机名） 和 domainname（域名） 两个系统标识
	//2.IPC Namespace 用来隔离 System V IPC 和 POSIX message queues(可以简单理解进程间通信用的消息队列)
	//3.PID Namespace 是用来隔离进程 ID 的
	//4.Mount Namespace 用来隔 离各个进程看到 的挂载点视图
	//5.User N amespace 主要是隔离用户 的 用A户组 ID 。
	//6.Network Namespace 是用来隔离网络设备、 IP 地址端口 等网络械的 Namespace

	//fork出来的新进程内的初始命令,默认使用sh来执行
	cmd := exec.Command("/proc/self/exe", args...)
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

	return cmd
}
