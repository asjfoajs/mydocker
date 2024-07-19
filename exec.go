package main

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"mydocker/container"
	_ "mydocker/nsenter" //重点只有导入这个包，那个c语言的才会被调用
	"os"
	"os/exec"
	"strings"
)

/*
前面的C代码里已经出现mydocker_pid和mydocker_cmd这两个key，主要是为了控制是否执行c代码里面的setns
*/
const ENV_EXEC_PID = "mydocker_pid"
const ENV_EXEC_CMD = "mydocker_cmd"

func ExecContainer(containerName string, comArray []string) {
	//根据传递过来的容器名获取宿主机对应的PID
	pid, err := getContainerPidByName(containerName)
	if err != nil {
		logrus.Errorf("Exec container getContainerPidByName %s error %v", containerName, err)
		return
	}

	//把命令以空格为分隔符拼接成一个字符串，便于传递
	cmdStr := strings.Join(comArray, " ")
	logrus.Infof("container pid %s", pid)
	logrus.Infof("command %s", cmdStr)

	//这里是重点
	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	//只有exec会设置这个值
	os.Setenv(ENV_EXEC_PID, pid)
	os.Setenv(ENV_EXEC_CMD, cmdStr)

	//获取对应PID的环境变量，其实也就是容器的环境变量
	containerEnvs := getEnvsByPid(pid)
	//将宿主机的环境变量和容器的环境变量都放置到exec进程内
	cmd.Env = append(os.Environ(), containerEnvs...)

	if err := cmd.Run(); err != nil {
		logrus.Errorf("Exec container %s error %v", containerName, err)
	}
}

// getContainerPidByName 这里是根据容器名获取对应容器的PID
func getContainerPidByName(containerName string) (string, error) {
	//先拼接出存储容器信息的路径
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	configFilePath := dirURL + container.ConfigName
	//读取该对应路径下的文件内容
	contentBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return "", err
	}
	var containerInfo container.ContainerInfo
	//将文件内容反序列化成容器信息对象，然后返回对应的PID
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return "", err
	}
	return containerInfo.Pid, nil
}

// getEnvsByPid 根据PID获取环境变量
func getEnvsByPid(pid string) []string {
	//进程环境变量存放的位置是 /proc/PID/environ
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := ioutil.ReadFile(path)
	if err != nil {
		logrus.Errorf("Read file %s error %v", path, err)
		return nil
	}
	//多个环境变量中的分割符是\u0000
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs

}
