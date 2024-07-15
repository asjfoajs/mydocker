package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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

	setUpMount()

	//打印一下cmdArray
	logrus.Infof("commandArray %s", cmdArray)

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

func pivotRoot(root string) error {
	//为了当前root的老root和新root不在同一个文件系统下，我们把root重新mount了一次，
	//bind mount是把相同的内容换了一个挂载点的挂载方法。
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("Mount rootfs to iteself error: %v", err)
	}

	//创建rootfs/.pivot_root 存储 old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}

	//pivot_root 到新的rootfs，老的old_root现在挂载在rootfs/.pivot_root上
	// 挂载点目前依然可以在mount命令中看到
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivot_root %v", err)
	}

	//修改当前的工作目录到根目录
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %v", err)
	}

	pivotDir = filepath.Join("/", ".pivot_root")
	// umount rootfs/.pivot_root
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir %v", err)
	}
	//删除临时目录文件夹
	return os.Remove(pivotDir)
}

/*
init 挂载点
*/
func setUpMount() {

	//获取当前路径
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Get current location error %v", err)
		return
	}

	logrus.Infof("Current location is %s", pwd)
	pivotRoot(pwd)

	//MS_NOEXEC 在本文件系统中不允许运行其他程序
	//MS_NOSUID 在本系统中运行程序的时候，不允许set-user-ID或set-group-ID
	//MS_NODEV 这个参数是从linux2.4以来，所有mount的系统都会默认设定的参数。
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	//mount proc
	err = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		logrus.Errorf("mount proc err %v", err)
		return
	}
	//不挂载 /dev，会导致容器内部无法访问和使用许多设备，这可能导致系统无法正常工作
	syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")

}
