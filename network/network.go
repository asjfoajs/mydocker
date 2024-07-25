package network

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"mydocker/container"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	//"github.com/vishvananda/netns"
)

var (
	defaultNetworkPath = "/var/run/mydocker/network/network/"
	drivers            = map[string]NetworkDriver{}
	networks           = map[string]*Network{}
)

// Network 网络
type Network struct {
	Name    string     //网络名
	IpRange *net.IPNet //地址段
	Driver  string     //网络驱动名
}

// Endpoint 网络端点
type Endpoint struct {
	ID          string           `json:"id"`
	Device      netlink.Veth     `json:"device"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	PortMapping []string         `json:"portmapping"`
	Network     *Network         `json:"network"`
}

// NetworkDriver 网络驱动
type NetworkDriver interface {
	Name() string                                          //网络驱动名
	Create(subnet string, name string) (*Network, error)   //创建网络
	Connect(network *Network, endpoint *Endpoint) error    //连接网络
	Disconnect(network *Network, endpoint *Endpoint) error //断开网络
	Delete(network *Network) error                         //删除网络
}

func Init() error {
	//加载网络驱动
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver

	//判断网络的配置目录是否存在，不存在则创建
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(defaultNetworkPath, 0644)
		} else {
			return err
		}
	}

	//检查网络配置目录中的所有文件
	//filepath.Walk(path, func(string,os.FileInfo,error))
	//并执行第二个参数中的函数指针去处理目录下的每一个文件
	filepath.Walk(defaultNetworkPath, func(nwPath string, info os.FileInfo, err error) error {
		//如果是目录则跳过
		if info.IsDir() {
			return nil
		}

		//加载文件名作为网络名
		_, nwName := path.Split(nwPath)
		nw := &Network{
			Name: nwName,
		}

		//调用前面介绍的Network.load方法加载网络的配置信息
		if err := nw.load(nwPath); err != nil {
			logrus.Errorf("error load network: %s", err)
		}

		//将网络的配置信息加入到networks字典中
		networks[nwName] = nw
		return nil
	})
	return nil
}

func ListNetwork() {
	//通过前面在mydocker ps时介绍的tabwreite的库去展示网络
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	//遍历网络信息
	for _, nw := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n", nw.Name, nw.IpRange.String(), nw.Driver)
	}
	//输出到标准输出
	if err := w.Flush(); err != nil {
		logrus.Errorf("Flush error %v", err)
		return
	}
}

// CreateNetwork 创建网络
func CreateNetwork(driver, subnet, name string) error {
	//ParseC是Golang net包的函数，功能是将网段的字符串转换成net.IPNet的对象
	_, cidr, _ := net.ParseCIDR(subnet)
	//通过IPAM分配网关IP，获取到网段中第一个IP作为网关的IP
	gatewayIp, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIp

	//调用地址定的网络驱动创建网络这里的drivers字典是各个网络驱动的实例字典，通过调用网络驱动
	//的Create方法创建网络，后面会以Bridge驱动为例介绍它的实现
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}

	//保存网络信息，将网络的信息保存在文件系统中，以便查询和在网络上连接网络端点
	return nw.dump(defaultNetworkPath)
}

func DeleteNetwork(networkName string) error {
	//查找网络是否存在
	nw, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such Network: %s", networkName)
	}

	//调用IPAM的实例ipAllocator的实例去释放网关IP
	if err := ipAllocator.Release(nw.IpRange, &nw.IpRange.IP); err != nil {
		return err
	}

	//调用网络驱动删除网络创建的设备于配置，后面会以Bridge驱动删除网络为例子介绍如何实现网络驱动删除网络
	if err := drivers[nw.Driver].Delete(nw); err != nil {
		return fmt.Errorf("Error Remove Network DriveError :%s", err)
	}

	//从网络的配置目录中删除该网络对应的配置文件
	return nw.remove(defaultNetworkPath)
}

// Content 容器连接到网络 mydocker run -net restnet -p 8080：80 xxxx
func Content(networkName string, cinfo *container.ContainerInfo) error {
	//从networks字典中取到容器连接的网络的信息，networks字典中保存了当前已经创建的网络
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("No Such Network:%s", networkName)
	}

	//通过调用IPAM从网络的网段中获取可用的IP作为容器IP地址
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}

	//创建网络端点
	ep := &Endpoint{
		ID:          fmt.Sprintf("%s-%s", cinfo.ID, networkName),
		IPAddress:   ip,
		Network:     network,
		PortMapping: cinfo.PortMapping,
	}

	//调用网络驱动的Connect方法去连接和配置网络端点
	//完成了1.创建Veth 2.挂载一端到 Bridge上
	if err = drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}

	//进入到容器的网络Namespace配置容器网络设备的IP地址和路由
	//完成3.将另一端移动到netns中4.设置另一端的IP地址5.设置netns中的路由
	if err = configEndpointIpAddressAndRoute(ep, cinfo); err != nil {
		return err
	}

	//配置容器到宿主机的端口映射， 例如mydocker run -p 8080:80 xxx
	//6.设置端口映射
	return configPortMapping(ep, cinfo)

}

func (nw *Network) remove(dumpPath string) error {
	//网络对应的配置文件，即配置目录下的的网络名文件
	//检查文件状态，如果文件已经不存在就直接返回
	if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
		os.MkdirAll(dumpPath, 0644)
		return os.Remove(path.Join(dumpPath, nw.Name))
	} else {
		return err
	}
}
func (nw *Network) dump(dumpPath string) error {
	//检查保存的目录是否存在，不存在则创建
	if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
		os.MkdirAll(dumpPath, 0644)
	} else {
		return err
	}

	//保存的文件名是网络的名字
	nwPath := path.Join(dumpPath, nw.Name)
	//打开保存的文件用于写入，后面打开的模式参数分别是存在内容则清空 只写入 不存在则创建
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logrus.Errorf("error:%v", err)
		return err
	}
	defer nwFile.Close()

	//通过json的库序列化网络对象到json的字符串
	nwJson, err := json.Marshal(nw)
	if err != nil {
		logrus.Errorf("error:%v", err)
		return err
	}

	//将网络配置的json字符串写入到文件中
	_, err = nwFile.Write(nwJson)
	if err != nil {
		logrus.Errorf("error:%v", err)
		return err
	}
	return nil
}

func (nw *Network) load(dumpPath string) error {
	//打开配置文件
	nwConfigFile, err := os.Open(dumpPath)
	defer nwConfigFile.Close()
	if err != nil {
		return err
	}

	//从配置文件中读取网络的配置json字符串
	nwJson := make([]byte, 2000)
	n, err := nwConfigFile.Read(nwJson)
	if err != nil {
		return err
	}

	//通过json字符串反序列化出网络
	err = json.Unmarshal(nwJson[:n], nw)
	if err != nil {
		logrus.Errorf("Error load nw info:%v", err)
		return err
	}
	return nil
}

// 配置容器Namespace中的网络设备及路由
// 就是将Veth另一端配置IP并激活绑定Namespace
// 并设置容器内部所有请求的路由都经过Veth
func configEndpointIpAddressAndRoute(ep *Endpoint, cinfo *container.ContainerInfo) error {
	//通过网络端点中“Veth”的另一端
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("fail config endpoint: %v", err)

	}

	//将容器的网络端点加入到容器的网络空间中
	//并使这个函数下面的操作都在这个网络空间中进行
	//执行完函数后，恢复为默认的网络空间
	defer enterContainerNetns(&peerLink, cinfo)()

	//获取到容器的IP地址及网段，用于配置容器内部几口地址
	//比如容器IP是192.168.1.2，而网络的网段是192.168.1.0/24
	//那么这里产出的IP字符串是192.168.1.2/24,用于容器内Veth断点配置
	interfaceIP := *ep.Network.IpRange
	interfaceIP.IP = ep.IPAddress

	//调用setInterfaceIP函数设置容器内Veth端点的IP
	//这个函数，在上一节配置Bridge时有介绍其实现
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v,%s", ep.Network, err)
	}

	//启动容器内的Veth端点
	if err = setInterfaceUp(ep.Device.PeerName); err != nil {
		return err
	}

	//Net Namespace中默认本地地址127.0.0.1的“lo”网卡是关闭状态的
	//启动它以保证容器访问自己的请求
	if err = setInterfaceUp("lo"); err != nil {
		return err
	}

	//设置容器内的外部请求都通过容器内的Veth端点访问
	// 0.0.0.0/0的网段，表示所有的IP地址段
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	//构建要添加的路由数据，包括网络设备 网管IP及目的网段
	//相当于route add -net 0.0.0.0/0 gw {Bridge网桥地址} dev {容器内的Veth断点设备}
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.IpRange.IP,
		Dst:       cidr,
	}

	//调用netlink的RouteAdd，添加路由到容器的网络空间
	//routeAdd函数相当于route add命令
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}

	return nil
}

// 将容器的网络端点加入到容器的网络空间中
// 并锁定当前程序所执行的线程，使当前线程进入到容器的网络空间
// 返回值是一个函数指针，执行这个返回函数才会退出容器的网络空间，回归到宿主机的网络空间
// 这个函数中引用了之前介绍的github.com/vishvananda/netns包来做Namespace操作
func enterContainerNetns(enLink *netlink.Link, cinfo *container.ContainerInfo) func() {
	//找到容器的Net Namespace
	// /proc/{pid}/ns/net 打开这个文件的文件描述符就可以来操作Net Namespace
	// 而ContainerInfo中的Pid，即容器在宿主机上的映射的进程ID
	// 它对应的/proc/{pid}/ns/net 就是容器内部的Net Namespace
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", cinfo.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("error get container net namespace,%v", err)
	}

	//取到文件的文件描述符
	nsFD := f.Fd()
	//锁定当前程序所执行的线程，如果不锁定操作系统线程的话
	//GO语言的goroutine可能会被调度到别的线程上去
	//就不能呢个保证一直在所需的网络空间中了
	//所以调用runtime.LockOSThread时要先锁定当前程序执行的线程
	runtime.LockOSThread()

	//修改网络端点Veth的另一端，将其移动到容器的Net Namespace中
	//做网络隔离
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		logrus.Errorf("error set link netns,%v", err)
	}
	//通过netns.Get方法，获取当前进程的Net Namespace
	//以便后面从容器的Net Namespace中退出，回到原本网络的Net Namespace中
	origns, err := netns.Get()
	if err != nil {
		logrus.Errorf("error get current netns,%v", err)
	}

	//将当前进程也加入进去
	//调用netns.Set方法，将当前进程加入容器的Net namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		logrus.Errorf("error set netns, %v", err)
	}

	//返回之前Net Namespace的函数
	//再容器的网络空间中，执行完容器配置之后调用此函数就可以将程序恢复到原生的Net Namespace
	return func() {
		//恢复到上面获取到的之前的Net Namespace
		netns.Set(origns)
		//关闭Namespace文件
		origns.Close()
		//取消对当前程序的线程锁定
		runtime.UnlockOSThread()
		//关闭Namespace文件
		f.Close()
	}

}

// 配置端口映射
func configPortMapping(ep *Endpoint, cinfo *container.ContainerInfo) error {
	//遍历容器端口映射列表
	for _, pm := range ep.PortMapping {
		//分割成宿主机的端口和容器的端口
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			logrus.Errorf("port mapping format error,%v", pm)
			continue
		}

		//由于iptables没有Go语言版本的实现，所以采用exec.Command的方式直接调用命令配置
		//在iptables的PREROUTING中添加DNAT规则
		//将宿主机的端口请求转发到容器的地址和端口上
		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAR --to-destination %s:%s",
			portMapping[0], ep.IPAddress.String(), portMapping[1])
		//执行iptables命令，添加端口映射转发规则
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptables Output,%v", output)
			continue
		}
	}
	return nil
}
