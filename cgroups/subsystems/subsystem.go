package subsystems

// ResourceConfig 用于传递资源限制配置的结构体，包含内存限制，cpu时间片权重，cpu核心数
type ResourceConfig struct {
	MemoryLimit string
	CpuShare    string
	CpuSet      string
}

// Subsystem ，每个Subsystem可以实现下面的4个接口
// 这里将cgroup抽象成了path，原因是cgroup在hierarchy的路径，便是虚拟文件系统中的虚拟路径
type Subsystem interface {
	// Name 返回subsystem的名字，比如cpu，memory
	//Name() string
	// Set 设置某个cgroup在这个subsystem中的资源限制
	Set(path string, res *ResourceConfig) error
	// Apply 将进程添加到某个cgroup中
	Apply(path string, pid int) error
	// Remove 移除某个cgroup
	Remove(path string) error
}

// SubsystemsIns 通过不同的subsystem初始化实例创建资源限制处理链数组
var (
	SubsystemsIns = []Subsystem{
		//&CpuSetSubsystem{},
		&MemorySubSystem{},
		//&CpuSubsystem{},
	}
)
