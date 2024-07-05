package example

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	//1.UTS Namespace 主要用来隔离 hostname（主机名） 和 domainname（域名） 两个系统标识
	//2.IPC Namespace 用来隔离 System V IPC 和 POSIX message queues(可以简单理解进程间通信用的消息队列)
	//3.PID Namespace 是用来隔离进程 ID 的
	//4.Mount Namespace 用来隔 离各个进程看到 的挂载点视图
	//5.User N amespace 主要是隔离用户 的 用A户组 ID 。
	//6.Network Namespace 是用来隔离网络设备、 IP 地址端口 等网络械的 Namespace

	//fork出来的新进程内的初始命令,默认使用sh来执行
	cmd := exec.Command("sh")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | //创建一个uts namespace
			syscall.CLONE_NEWIPC | //创建一个ipc namespace
			syscall.CLONE_NEWPID | //创建一个pid namespace
			syscall.CLONE_NEWNS | //创建一个mount namespace
			syscall.CLONE_NEWUSER | //创建一个user namespace
			syscall.CLONE_NEWNET, // 创建一个network namespace
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
