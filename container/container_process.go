package container

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

var (
	RUNNING             string = "running"
	STOP                string = "stopped"
	EXIT                string = "exited"
	DefaultInfoLocation string = "/var/run/mydocker/%s/"
	ConfigName          string = "config.json"
	ContainerLogFile    string = "container.log"
)

type ContainerInfo struct {
	Pid         string   `json:"pid"`          //容器的init进程在宿主机上的PID
	ID          string   `json:"id"`           //容器ID
	Name        string   `json:"name"`         //容器名
	Command     string   `json:"command"`      //容器内init进程的运行命令
	CreatedTime string   `json:"created_time"` //创建时间
	Status      string   `json:"status"`       //容器的状态
	Volume      string   `json:"volume"`       //容器挂载的数据卷
	PortMapping []string `json:"portmapping"`  //端口映射
}

/*
这里是父进程，也就是当前进程执行的内容，根据上一章介绍的内容，应该比较容易明白
1.这里的/proc/self/exe调用中，/proc/self/指的是当前运行进程的自己的环境
，exec其实就是自己调用了自己，使用这种方式对创建出来的进程进行初始化
2.后面args是参数，其中init是传递给本进程的第一个参数，在本例中，其实就是会去调用initCommand去初始化
进程的一些环境和资源
3.下面的clone参数就是去fork出来一个新进程，并且使用了namespace隔离创建的进程和外部环境。
4.如果用户指定了 -ti 参数，就需要把当前进程的输入输出导入到标准输入输出上
*/
func NewParentProcess(tty bool, volume, containerName, imageName string, envSlice *[]string) (*exec.Cmd, *os.File) {
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

	//因为peer group和propagate type(传播属性)所以要先设置成private并递归
	syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, "")

	//busybox目录
	//cmd.Dir = "/root/busybox"

	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		//生成容器对应目录的container.log文件
		dirURL := fmt.Sprintf(DefaultInfoLocation, containerName)
		if err := os.MkdirAll(dirURL, 0622); err != nil {
			logrus.Errorf("NewParentProcess mkdir %s error %v", dirURL, err)
			return nil, nil
		}
		stdLogFilePath := dirURL + ContainerLogFile
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			logrus.Errorf("NewParentProcess create %s error %v", stdLogFilePath, err)
			return nil, nil
		}
		//把生成好的文件赋值给stdout,这样就能把容器内的标准输出重定向到这个文件中
		cmd.Stdout = stdLogFile

	}

	//注意，改动在这里，在这个地方传入管道文件读取端的句柄
	//这个属性的意思是会外带着这个文件句柄去创建子进程。为什么叫“外带着” 呢?
	//因为1个进程默认会有3个文件描述符,分别是标准输入、标准输出、标准错误。这3个是子进程一
	//创建的时候就会默认带着的,那么外带的这个文件描述符理所当然地就成为了第4个。
	cmd.ExtraFiles = []*os.File{readPipe}

	cmd.Env = append(os.Environ(), *envSlice...)

	NewWorkSpace(volume, imageName, containerName)
	cmd.Dir = GetMerge(containerName)

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

func NewWorkSpace(volume, imagerName, containerName string) {
	CreateLowerLayer(containerName, imagerName)
	CreteUpperLayer(containerName)
	CreteWorkLayer(containerName)
	CreateMountPoint(containerName)

	//根据volume判断是否执行挂在数据卷的操作
	if volume != "" {
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			MountVolume(containerName, volumeURLs)
			logrus.Infof("%q", volumeURLs)
		} else {
			logrus.Infof("Volume parameter input is not correct")
		}

	}

}

// CreateLowerLayer 将busybox.tar解压到busybox目录下，作为容器的只读层
func CreateLowerLayer(containerName, imageName string) {
	lowerPath := GetLower(containerName)
	imagePath := GetImage(imageName)
	exists, err := PathExists(lowerPath)

	if err != nil {
		logrus.Infof("Fail to judge whether dir %s exists., %v", lowerPath, err)
	}

	if exists == false {
		if err := os.MkdirAll(lowerPath, 0777); err != nil {
			logrus.Infof("Fail to create dir %s, %v", lowerPath, err)
		}

		if _, err := exec.Command("tar", "-xvf", imagePath, "-C", lowerPath).CombinedOutput(); err != nil {
			logrus.Errorf("untTar dir %s error %v", imageName, err)
		}

	}

}

// CreteUpperLayer 创建一个名为upper的文件夹作为容器唯一的可写层
func CreteUpperLayer(containerName string) {
	upperPath := GetUpper(containerName)
	if err := os.Mkdir(upperPath, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", upperPath, err)
	}
}

func CreteWorkLayer(containerName string) {
	workPath := GetWorker(containerName)
	if err := os.Mkdir(workPath, 0777); err != nil {
		logrus.Errorf("Mkdir dir %s error. %v", workPath, err)
	}
}

func CreateMountPoint(containerName string) {
	mergePath := GetMerge(containerName)
	//创建mnt文件夹作为挂载点
	if err := os.Mkdir(mergePath, 0777); err != nil {
		logrus.Infof("Mkdir dir %s error. %v", mergePath, err)
	}

	//把writeLayer目录和busybox目录mount到mnt目录下
	//dirs := "dirs=" + rootURL + "writeLayer" + rootURL + "busybox"
	//cmd := exec.Command("mount", "-t", "aufs", "-0", dirs, "none", mntURL)

	//因为用的overlay2，还需要一个work层
	//sudo mount -t overlay -o lowerdir=image-layer,upperdir=container-layer,workdir=work none mnt
	//writeURL := rootURL + "/writeLayer"
	//readOnlyURL := rootURL + "/busybox"
	cmd := exec.Command("mount", "-t", "overlay",
		"-o", GetOverlayFSDirs(GetLower(containerName), GetUpper(containerName), GetWorker(containerName)),
		"none", mergePath)
	//cmd := exec.Command("mount", "-t", "overlay", "-o", "lowerdir=", readOnlyURL, ",upperdir="+writeURL, ",workdir=", workURL, "none", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
	}
}

func DeleteWorkSpace(volume, containerName string) {
	//如果指定了volume, 则写在挂载点
	if volume != "" {
		volumeURLs := volumeUrlExtract(volume)
		length := len(volumeURLs)
		if length == 2 && volumeURLs[0] != "" && volumeURLs[1] != "" {
			DeleteMountPointWithVolume(containerName, volumeURLs)
		} else {
			DeleteMountPoint(containerName)
		}
	} else {
		DeleteMountPoint(containerName)
	}
	DeleteUpperLayer(containerName)
	DeleteWorkLayer(containerName)
}

func DeleteMountPoint(containerName string) {
	mergePath := GetMerge(containerName)
	cmd := exec.Command("umount", mergePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("%v", err)
	}
	if err := os.RemoveAll(mergePath); err != nil {
		logrus.Errorf("Remove dir %s error %v", mergePath, err)
	}

}

func DeleteUpperLayer(containerName string) {
	upperPath := GetUpper(containerName)
	if err := os.RemoveAll(upperPath); err != nil {
		logrus.Errorf("Remove dir %s error %v", upperPath, err)
	}
}

func DeleteWorkLayer(containerName string) {
	workPath := GetWorker(containerName)
	if err := os.RemoveAll(workPath); err != nil {
		logrus.Errorf("Remove dir %s error %v", workPath, err)
	}
}

// MountVolume 挂载数据卷就三步：1.创宿主机的目录2.创容器的目录3.挂载
func MountVolume(containerName string, volumeURLs []string) {
	//创建宿主机文件目录,不存在会创建一下
	parentUrl := volumeURLs[0]
	if err := createFile(parentUrl); err != nil {
		logrus.Errorf("Create parent dir %s error. %v", parentUrl, err)
		return
	}

	//在容器文件系统里创建挂载点
	containerUrl := volumeURLs[1]
	containerVolumeURL := GetMerge(containerName) + containerUrl
	if err := createFile(containerVolumeURL); err != nil {
		logrus.Errorf("Create parent dir %s error. %v", parentUrl, err)
		return
	}
	//if err := os.Mkdir(containerVolumeURL, 0777); err != nil {
	//	logrus.Infof("Mkdir container dir %s error. %v", containerVolumeURL, err)
	//}

	////把宿主机文件目录挂载到容器挂载点
	//dirs := "dirs=" + parentUrl
	//cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", containerVolumeURL)

	////lowerdir=%s,upperdir=%s,workdir=%s
	//workURL := parentUrl + "Work"
	//CreteWorkDir(workURL)
	//cmd := exec.Command("mount", "-t", "overlay",
	//	"-o", fmt.Sprintf("lowerdir=%s,workdir=%s", parentUrl, workURL),
	//	"none", containerVolumeURL)

	cmd := exec.Command("mount", "-o", "bind", parentUrl, containerVolumeURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Mount volume failed. %v", err)
	}
}

func DeleteMountPointWithVolume(containerName string, volumeURLS []string) {
	mergePath := GetMerge(containerName)
	//卸载容器里volume挂载点的文件系统
	containerUrl := mergePath + volumeURLS[1]
	cmd := exec.Command("umount", containerUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Umount volume failed. %v", err)
	}

	//卸载整个容器文件系统的挂载点
	cmd = exec.Command("umount", mergePath)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		logrus.Errorf("Umount volume failed. %v", err)
	}

	//删除容器文件系统的挂载点
	if err := os.RemoveAll(mergePath); err != nil {
		logrus.Infof("Remove mountpoint dir %s error %v", mergePath, err)
	}

	//workURL := volumeURLS[0] + "Work"
	//DeleteWorkLayer(workURL)
}

// PathExists 判断文件的路径是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path) //文件是否可读
	if err == nil {
		return true, err
	}

	if os.IsNotExist(err) { //如果文件不存在，返回false
		return false, nil
	}

	return false, err //存在但不可访问
}

func volumeUrlExtract(volume string) []string {
	return strings.Split(volume, ":")
}

func createFile(path string) error {
	//创建宿主机文件目录,不存在会创建一下
	//parentUrl := volumeURLs[0]
	exists, err := PathExists(path)
	if err != nil {
		return fmt.Errorf("fail to judge whether dir %s exists., %v", path, err)
	}

	if !exists {
		if err := os.Mkdir(path, 0777); err != nil {
			return fmt.Errorf("mkdir parent dir %s error. %v", path, err)
		}
	}

	return nil
}
