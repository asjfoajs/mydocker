package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"mydocker/cgroups"
	"mydocker/cgroups/subsystems"
	"mydocker/container"
	"mydocker/network"
	"mydocker/utils"
	"os"
	"strconv"
	"strings"
	"time"
)

// 将RootUrl，MntUrl和WriteLayer添加为全局变量
// var (
//
//	RootURL       string = "/root"
//	MntURL        string = "/root/mnt/%s"
//	WriteLayerURL string = "/root/writeLayer/%s"
//	WorkURL       string = "/root/worker/%s"
//
// )

/*
*这里的Start方法是真正开始前面创建好的command的调用，它首先会clone出来一个namespace隔离的
进程，然后再子进程中，调用/proc/self/exe，也就是调用自己，发送init参数，调用我们写的init方法，去初始化容器的一些资源。
*/
func Run(tty bool, volume, containerName, imageName, net string, comArray, envSlice, portMapping *[]string, res *subsystems.ResourceConfig) {
	//首先生成10位容器ID
	containerId := utils.RanStringBytes(10)
	//如果用户不指定容器名，那么就以容器id当作容器名
	if containerName == "" {
		containerName = containerId
	}
	//logrus.Infof("Run command %s", command)
	parent, wirtePipe := container.NewParentProcess(tty, volume, containerName, imageName, envSlice)
	if parent == nil {
		logrus.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		logrus.Error("start parent process error %v", err)
	}

	//记录容器信息
	containerInfo, err := recordContainerInfo(parent.Process.Pid, containerId, containerName, volume, comArray, portMapping)
	if err != nil {
		logrus.Errorf("record container info error %v", err)
		return
	}

	//创建cgroup manager，并通过调用set和apply设置资源限制并使限制在容器上生效
	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup")

	//defer cgroupManager.Destroy()

	//设置资源限制
	cgroupManager.Set(res)
	//将容器进程加入到各个subsystem挂载对应的cgroup中
	cgroupManager.Apply(parent.Process.Pid)

	//如果指定了网络信息则进行配置
	if net != "" {
		//创建网络配置文件
		//containerInfo := &container.ContainerInfo{
		//	Pid:         strconv.Itoa(parent.Process.Pid),
		//	ID:          containerId,
		//	Name:        containerName,
		//	PortMapping: *portMapping,
		//}

		err := network.Content(net, containerInfo)
		if err != nil {
			logrus.Errorf("Error Connect Network %v", err)
			return
		}
	}

	//对容器设置完限制之后初始化容器
	sendInitCommand(comArray, wirtePipe)

	if tty {
		parent.Wait()

		deleteContainerInfo(containerName)

		//卸载并删除
		//mntURL := "/root/mnt"
		//workURL := "/root/worker"
		//rootURL := "/root"

		container.DeleteWorkSpace(volume, containerName)

		os.Exit(0)
	}

	//time.Sleep(2 * time.Minute)
}

func sendInitCommand(comArray *[]string, writePipe *os.File) {
	command := strings.Join(*comArray, " ")
	writePipe.WriteString(command)
	writePipe.Close()
}

func recordContainerInfo(containerPID int, containerId, containerName, volume string, commandArray, portMapping *[]string) (*container.ContainerInfo, error) {
	////首先生成10位容器ID
	//id := utils.RanStringBytes(10)
	//以当前时间为容器创建时间
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	command := strings.Join(*commandArray, "")
	////如果用户不指定容器名，那么就以容器id当作容器名
	//if containerName == "" {
	//	containerName = id
	//}

	//生成容器信息的结构体实例
	containerInfo := &container.ContainerInfo{
		ID:          containerId,
		Pid:         strconv.Itoa(containerPID),
		Command:     command,
		CreatedTime: currentTime,
		Status:      container.RUNNING,
		Name:        containerName,
		Volume:      volume,
		PortMapping: *portMapping,
	}

	//将容器信息的对象json序列化成字符串
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		logrus.Errorf("Record container info error %v", err)
		return nil, err
	}
	jsonStr := string(jsonBytes)

	//拼凑一下存储容器信息的路径
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	//如果该路径不存在，就级联地全部创建
	if err = os.MkdirAll(dirUrl, 0622); err != nil {
		logrus.Errorf("mkdir container info path %s error %v", dirUrl, err)
		return nil, err
	}

	fileName := dirUrl + "/" + container.ConfigName
	//创建最终的配置文件config.json文件
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		logrus.Errorf("Create file %s error %v", fileName, err)
		return nil, err
	}

	//将json序列化后的字符串写入到文件中
	if _, err := file.WriteString(jsonStr); err != nil {
		logrus.Errorf("File write string error %v", err)
	}

	return containerInfo, nil
}

func deleteContainerInfo(containerName string) {
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirURL); err != nil {
		logrus.Errorf("Remove dir %s error %v", dirURL, err)
	}
}
