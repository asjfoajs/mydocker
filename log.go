package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"mydocker/container"
	"os"
)

func logContainer(containerName string) {
	//找到对应文件夹的位置
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	logFileLocation := dirURL + container.ContainerLogFile
	//打开日志文件
	file, err := os.Open(logFileLocation)
	defer file.Close()
	if err != nil {
		logrus.Errorf("Log container open file %s error %v", logFileLocation, err)
		return
	}
	//将文件内的内容都读取出来
	content, err := ioutil.ReadAll(file)
	if err != nil {
		logrus.Errorf("Log container read file %s error %v", logFileLocation, err)
		return
	}
	//使用fmt.Fprintf将内容输入到标准输出，也就是控制台上
	fmt.Fprintf(os.Stdout, "%s", content)
}
