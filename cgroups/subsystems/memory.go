package subsystems

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

// MemorySubSystem  memory subsystem的实现
type MemorySubSystem struct {
}

// Set 设置cgroupPath对应的cgroup的内存资源限制
func (s *MemorySubSystem) Set(cgroupPath string, res *ResourceConfig) error {
	//GetCgroupPath 的作用是获取当前subsystem在虚拟文件系统中的路径，GetCgroupPath这个函数在下面会介绍。
	if subsysCgroupPath, err := GetCgroupPath(cgroupPath, true); err == nil {
		if res.MemoryLimit != "" {
			//设置这个cgroup的内存限制，即将限制写入到cgroup对应目录的” memory.limit_in_bytes,新版本是memory.max文件中。
			if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "memory.max"), []byte(res.MemoryLimit), 0644); err != nil {
				return fmt.Errorf("set cgroup memory fail %v", err)
			}
		}
		return nil
	} else {
		return err
	}
}

// Remove 删除cgroupPath对应的cgroup
func (s *MemorySubSystem) Remove(cgroupPath string) error {
	if subsysCgroupPath, err := GetCgroupPath(cgroupPath, false); err == nil {
		//删除cgroup便是删除对应的cgroupPath的目录
		return os.Remove(subsysCgroupPath)
	} else {
		return err
	}
}

// Apply 将一个进程假如到cgroupPath对应的cgroup中
func (s *MemorySubSystem) Apply(cgroupPath string, pid int) error {
	if subsysCgroupPath, err := GetCgroupPath(cgroupPath, false); err == nil {
		//把进程的PID写到cgroup的虚拟文件系统对应目录下的"task"文件中，新版本是cgroup.procs
		if err := ioutil.WriteFile(path.Join(subsysCgroupPath, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0644); err != nil {
			return fmt.Errorf("set cgroup proc fail %v", err)
		}
		return nil
	} else {
		return fmt.Errorf("get cgroup %s error :%v", cgroupPath, err)
	}
}

//// cgroupV2已经不需要了 因为在cgroupV2中，cgroup的目录结构发生了改变meory，cpu等在同一文件下
//// Name 返回cgroup的名字
//func (s *MemorySubSystem) Name() string {
//	return "memory"
//}
