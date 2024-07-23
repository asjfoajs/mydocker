package network

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"mydocker/container"
	"net"
	"os"
	"path"
	"path/filepath"
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

	//调用网络驱动的Connect方法去链接和配置网络端点，后面会以Bridge驱动为例介绍它的实现
	if err = drivers[network.Driver].Connect(network, ep); err != nil {
		return err
	}

	//进入到容器的网络Namespace配置容器网络设备的IP地址和路由，具体实现也会在“Bridge”网络驱动的实现中介绍
	if err = configEndpointIpAddressAndRoute(ep, cinfo); err != nil {
		return err
	}

	//配置容器到宿主机的端口映射，具体实现也会在“Bridge”后面链接容器网络时会介绍
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
		logrus.Errorf("error:", err)
		return err
	}
	defer nwFile.Close()

	//通过json的库序列化网络对象到json的字符串
	nwJson, err := json.Marshal(nw)
	if err != nil {
		logrus.Errorf("error:", err)
		return err
	}

	//将网络配置的json字符串写入到文件中
	_, err = nwFile.Write(nwJson)
	if err != nil {
		logrus.Errorf("error:", err)
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
		logrus.Errorf("Error load nw info:", err)
		return err
	}
	return nil
}
