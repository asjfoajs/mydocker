package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"mydocker/container"
	"os"
)

func removeContainer(containerName string) {
	//根据容器名获取容器对应的信息
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {
		logrus.Errorf("get container info by name %s error %v", containerName, err)
		return
	}

	//只删除处于停止状态的容器
	if containerInfo.Status != container.STOP {
		logrus.Errorf("container %s is not stopped", containerName)
		return
	}
	//deleteContainerInfo(containerName)
	//找到对应存储容器信息的文件路径
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	//将所有信息包括子目录都删除
	if err := os.RemoveAll(dirURL); err != nil {
		logrus.Errorf("Remove container %s error %v", dirURL, err)
		return
	}

	//卸载和解绑
	container.DeleteWorkSpace(containerInfo.Volume, containerName)

}
