package main

import (
	"github.com/sirupsen/logrus"
	"mydocker/container"
	"os/exec"
)

func CommitContainer(containerName, imageName string) {
	//mntURL := "/root/mnt"
	//imageTar := "/root/" + imageName + ".tar"
	//fmt.Printf("%s", imageTar)
	mntURL := container.GetMerge(containerName)
	imageTar := container.GetImage(imageName)
	if _, err := exec.Command("tar", "-czf", imageTar, "-C", mntURL, ".").CombinedOutput(); err != nil {
		logrus.Errorf("Tar folder %s error %v", mntURL, err)
	}
}
