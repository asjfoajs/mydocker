package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"mydocker/container"
	"strconv"
	"syscall"
)

func stopContainer(containerName string) {
	//根据容器名获取对应的主进程PID
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		logrus.Errorf("get container pid by name %s error %v", containerName, err)
		return
	}

	//将string类型的PID转换为int类型
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		logrus.Errorf("atoi %s error %v", pid, err)
		return
	}

	//系统调用kill可以发送信号给进程，通过传递syscall.SIGTERM信号，去杀掉容器主进程
	if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
		logrus.Errorf("kill process %d error %v", pidInt, err)
		return
	}
	//根据容器名称获取对应的信息对象
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {
		logrus.Errorf("get container info by name %s error %v", containerName, err)
		return
	}

	//至此容器进程已经被kill，所以下面需要修改容器状态，PID可以置为空
	containerInfo.Status = container.STOP
	containerInfo.Pid = " "
	//将修改后的信息序列化成json字符串
	newContentBytes, err := json.Marshal(containerInfo)
	if err != nil {
		logrus.Errorf("get container info by name %s error %v", containerName, err)
		return
	}

	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName
	//重新写入新的数据覆盖原来的信息
	if err := ioutil.WriteFile(configFilePath, newContentBytes, 0622); err != nil {
		logrus.Errorf("write file %s error %v", configFilePath, err)
	}
}

// 根据容器名或获取对应的struct结构
func getContainerInfoByName(containerName string) (*container.ContainerInfo, error) {
	//构造存放容器信息的路径
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		logrus.Errorf("read file %s error %v", configFilePath, err)
		return nil, err
	}

	var containerInfo container.ContainerInfo
	//将容器信息字符串反序列化成对应的对象
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		logrus.Errorf("unmarshal config file %s error %v", configFilePath, err)
		return nil, err
	}
	return &containerInfo, err

}
