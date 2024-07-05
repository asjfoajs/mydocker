package example

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"syscall"
	"time"
)

const cgroupHierarchyMount = "/sys/fs/cgroup"

func main() {
	if os.Args[0] == "/proc/self/exe" {
		//容器进程
		time.Sleep(10 * time.Second) //先睡眠10等父进程写完cgroup在启动。
		fmt.Printf("current pid %d", syscall.Getpid())
		fmt.Println()
		cmd := exec.Command("sh", "-c", `stress --vm-bytes 200m --vm-keep -m 1`)
		cmd.SysProcAttr = &syscall.SysProcAttr{}

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

	} else {
		cmd := exec.Command("/proc/self/exe")
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		}

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			fmt.Println("ERROR", err)
			os.Exit(1)
		} else {
			//得到fork出来进程映射在外部命名空间的pid
			fmt.Printf("%v", cmd.Process.Pid)
			fmt.Println()

			hierarchyCgroupName := "testCgroup-" + strconv.Itoa(cmd.Process.Pid)
			//在系统默认创建挂载了 subsystem 的Hierarchy上创建cgroup
			os.Mkdir(path.Join(cgroupHierarchyMount, hierarchyCgroupName), 0755)
			//将容器进程加入到这个cgroup中
			ioutil.WriteFile(path.Join(cgroupHierarchyMount, hierarchyCgroupName, "cgroup.procs"), []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
			//限制cgroup进程使用
			ioutil.WriteFile(path.Join(cgroupHierarchyMount, hierarchyCgroupName, "memory.max"), []byte("100m"), 0644)
		}
		cmd.Process.Wait()
	}
}
