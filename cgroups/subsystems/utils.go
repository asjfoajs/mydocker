package subsystems

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
)

var (
	cgroupRootPath string       // 保存挂载点
	mu             sync.RWMutex // 互斥锁，用于线程安全
)

// FindCgroupMountpoint 通过 /proc/self/mountinfo找出挂载了某个subsysteam的hierarchy cgroup根节点所在的目录
// 这里已经改成获取cgroup2的挂载点
func FindCgroupMountpoint() string {
	mu.RLock()
	if cgroupRootPath != "" {
		mu.RUnlock()
		return cgroupRootPath // 如果已经找到过，就不再重复查找
	}

	mu.RUnlock()
	mu.Lock()
	defer mu.Unlock()
	//在加入一下判断
	if cgroupRootPath != "" {
		return cgroupRootPath // 如果已经找到过，就不再重复查找
	}

	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		for _, field := range fields {
			//fmt.Println(opt)
			if field == "cgroup2" {
				cgroupRootPath = fields[4]
				return cgroupRootPath
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return ""
	}
	return ""
}

// GetCgroupPath 得到cgroup在文件系统中的绝对路径
func GetCgroupPath(cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot := FindCgroupMountpoint() // 获取cgroup挂载点 不用获取单个了
	if _, err := os.Stat(path.Join(cgroupRoot, cgroupPath)); err == nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path.Join(cgroupRoot, cgroupPath), 0755); err == nil {

			} else {
				return "", fmt.Errorf("error create cgroup %v", err)
			}
		}
		return path.Join(cgroupRoot, cgroupPath), nil
	} else {
		return "", fmt.Errorf("cgroup path error %v", err)
	}
}
