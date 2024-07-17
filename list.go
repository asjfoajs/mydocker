package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"mydocker/container"
	"os"
	"text/tabwriter"
)

func ListContainers() {
	//找到存储容器信息的路径/var/run/mydocker
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, "")
	dirURL = dirURL[:len(dirURL)-1]

	//读取该文件下的所有文件
	files, err := ioutil.ReadDir(dirURL)
	if err != nil {
		logrus.Errorf("Read dir %s error %v", dirURL, err)
		return
	}

	var containers []*container.ContainerInfo
	//遍历该文件夹下的所有文件
	for _, file := range files {
		//根据容器配置文件获取对应的信息，然后转换成容器信息的对象
		tmpContainer, err := getContainerInfo(file)
		if err != nil {
			logrus.Errorf("Get container info error %v", err)
			continue
		}

		containers = append(containers, tmpContainer)
	}

	//使用tabwriter.NewWrite在控制台打印出容器信息
	//tabwriter是引用的text/tabwrite类库，用于在控制台打印对齐的表格
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	//控制台输出的信息列
	fmt.Fprintf(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", item.ID, item.Name, item.Pid, item.Status, item.Command, item.CreatedTime)
	}

	//刷新标准输入流缓存区，将容器列表打印出来
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error %v", err)
		return
	}
}

func getContainerInfo(file os.FileInfo) (*container.ContainerInfo, error) {
	//获取文件名
	containerName := file.Name()
	//根据文件名生成文件绝对路径
	configFileDir := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFileDir = configFileDir + container.ConfigName
	//读取config.json文件内的容器信息
	content, err := ioutil.ReadFile(configFileDir)
	if err != nil {
		logrus.Errorf("Read file %s error %v", configFileDir)
		return nil, err
	}

	var containerInfo container.ContainerInfo
	//将json文件信息反序列化成容器信息对象
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		logrus.Errorf("Json unmarshal error %v", err)
		return nil, err
	}

	return &containerInfo, nil

}
