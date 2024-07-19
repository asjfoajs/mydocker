package container

import "fmt"

const (
	ImagePath       = "/var/lib/mydocker/image/"
	RootPath        = "/var/lib/mydocker/overlay2/"
	lowerDirFormat  = RootPath + "%s/lower"
	upperDirFormat  = RootPath + "%s/upper"
	workDirFormat   = RootPath + "%s/work"
	mergeDirFormat  = RootPath + "%s/merged"
	overlayFSFormat = "lowerdir=%s,upperdir=%s,workdir=%s"
)

// GetRoot 获取容器的根目录
func GetRoot(containerName string) string {
	return RootPath + containerName
}

// GetImage 获取镜像的绝对路径
func GetImage(imageName string) string {
	return fmt.Sprintf("%s/%s.tar", ImagePath, imageName)
}

func GetLower(containerName string) string {
	return fmt.Sprintf(lowerDirFormat, containerName)
}

func GetUpper(containerName string) string {
	return fmt.Sprintf(upperDirFormat, containerName)
}

func GetWorker(containerName string) string {
	return fmt.Sprintf(workDirFormat, containerName)
}
func GetMerge(containerName string) string {
	return fmt.Sprintf(mergeDirFormat, containerName)
}
func GetOverlayFSDirs(lower, upper, work string) string {
	return fmt.Sprintf(overlayFSFormat, lower, upper, work)
}
