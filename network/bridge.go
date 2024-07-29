package network

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
	"os/exec"
	"strings"
)

type BridgeNetworkDriver struct {
}

// 初始化Bridge设备
func (d *BridgeNetworkDriver) initBridge(n *Network) error {
	//1.创建Bridge虚拟设备
	bridgeName := n.Name
	if err := createBridgeInterface(bridgeName); err != nil {
		return fmt.Errorf("Error add bridge: %s,Error:%v", bridgeName, err)
	}

	//2.设置Bridge设置的地址和路由
	gatewayIP := *n.IpRange
	gatewayIP.IP = n.IpRange.IP
	if err := setInterfaceIP(bridgeName, gatewayIP.String()); err != nil {
		return fmt.Errorf("Error assiging address : %s on brifge: %s with an error of: %v", gatewayIP, bridgeName, err)
	}

	//3.启动Bridge设备
	if err := setInterfaceUp(bridgeName); err != nil {
		return fmt.Errorf("Eroor set bridge up: %s,Error :%v", bridgeName, err)
	}

	//4.设置iptabels的SNAT规则
	if err := setupIPTables(bridgeName, n.IpRange); err != nil {
		return fmt.Errorf("Error setting iptables: %v", err)
	}
	return nil
}

// 创建linux Bridge虚拟设备，就是创建网桥相当于brctl addbr name
// 不过这里用的是 ip link add name
func createBridgeInterface(bridgeName string) error {
	//先检查是否已经存在了这个同名的Bridge设备
	_, err := net.InterfaceByName(bridgeName)
	//如果已经存在或者报错则返回创建错误
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	//初始化一个netlink的Link基础对象，Link的名字即Brigde虚拟设备的名字
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName

	//使用刚才创建的Link的属性创建netlink的Bridge对象
	br := &netlink.Bridge{LinkAttrs: la}
	//调用netlink的LinkAdd方法创建Bridge设备
	//netlink的Linkadd方法是用来创建虚拟网络设备的，相当于ip link add xxxx
	if err := netlink.LinkAdd(br); err != nil {
		return fmt.Errorf("bridge creation failed for bridge %s:%v", bridgeName, err)
	}
	return nil

}

// 设置Bridge设备的地址和路由，例如setInterfaceIP("testbridge","192.168.0.1/24")
func setInterfaceIP(name string, rawIP string) error {
	//通过netlink的LinkByName方法找到需要设置得网络接口
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("error get interface:%v", err)
	}

	//由于netlink.ParseIPNet是对net.ParseCIDR的一个封装，因此可以将net.ParseCiDR的返回值中的IP和net整合

	//返回值中的ipNet既包含了网段的信息，192.168.0.0/24，也包含了原始的ip 192.168.0.1
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}

	//通过netlink.AddrAdd给网络接口配置地址，相当于ip addr add xxx dev xxx的命令
	//同时如果配置了地址所在网段的信息，例如192.168.0.0/24
	//还会配置路由表192.168.0.0/24转发到这个testbridge的网络接口上
	addr := &netlink.Addr{
		IPNet: ipNet, // 使用命名字段并转换ipNet为字符串
		Label: "",    // 提供缺少的Label字段的值
		Flags: 0,     // 提供Flags字段的值
		Scope: 0,     // 提供Scope字段的值
	}

	//logrus.Infoln("set interface ip:%v", addr)

	return netlink.AddrAdd(iface, addr)

}

// 启动Bridge设置，设置网络接口为up状态
func setInterfaceUp(name string) error {
	//通过netlink的LinkByName方法找到需要设置得网络接口
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("error get interface:%v", err)
	}

	//通过netlink的LinkSetUp方法设置网络接口为up状态
	//等价于ip link set xxx up 命令
	err = netlink.LinkSetUp(iface)
	if err != nil {
		return fmt.Errorf("error set interface up:%v", err)
	}
	return nil
}

// 设置iptabels linux Bridge SNAT规则
func setupIPTables(bridgeName string, subNet *net.IPNet) error {
	//由于go语言没有直接操控iptables操作的库，所以需要通过调用iptables命令来设置iptables规则
	//iptables -t nat -A POSTROUTING -s <bridgeName> ! -o <bridgeName> -j MASQUERADE
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subNet.String(), bridgeName)

	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error set iptables:%v", err)
	}
	return nil
}

func (d *BridgeNetworkDriver) Name() string {
	return "bridge"
}

// Create 根据子网信息创建Bridge设备并初始化
func (d *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	//TODO implement me
	//通过net包中的net.ParseCIDR方法，取到网段的字符串中的网关IP地址和网络IP段
	ip, ipRange, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, err
	}
	ipRange.IP = ip
	//初始化网络对象
	n := &Network{
		Name:    name,
		IpRange: ipRange,
		Driver:  d.Name(),
	}
	//配置Linux Bridge
	err = d.initBridge(n)
	if err != nil {
		logrus.Errorf("error init bridge:%v", err)
	}
	//返回配置好的网络
	return n, err

}

// Connect 连接一个网络和网络端点
// 创建一对Veth，将一端绑定到网桥并激活
func (d *BridgeNetworkDriver) Connect(network *Network, endpoint *Endpoint) error {
	//获取网络名，即Linux Bridge的名字
	bridgeName := network.Name
	//通过接口明获取到Linux Bridge接口的对象和接口属性
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("error get bridge:%v", err)
	}

	//创建Veth接口的配置
	la := netlink.NewLinkAttrs()
	//由于Linux几口明的限制，名字去endpoint ID的前5位
	la.Name = endpoint.ID[:5] //就是容器ID的前5位
	//通过设置Veth接口的master属性，设置这个Veth的一端挂载到到网络对应的Linux Bridge上
	la.MasterIndex = br.Attrs().Index

	//创建Veth对象，通过PeerName属性，设置Veth的另外一端的接口名
	//配置Veth另外一段的名字cif-{endpoint ID的前5位}
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.ID[:5],
	}

	//调用netlink的LinkAdd方法创建Veth接口
	//因为上面指定了link的MasterIndex是网络对象的Linux Bridge
	//所以Veth的一段就已经挂在到了网络对应的Linux Bridge上
	if err := netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("error add veth pair:%v", err)
	}

	//调用netlink的LinkSetUp方法设置Veth接口为up状态
	//相当于ip link set xxx up 命令
	if err = netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("Error set Endpoint Device up:%v", err)
	}

	return nil
}

func (d *BridgeNetworkDriver) Disconnect(network *Network, endpoint *Endpoint) error {
	//TODO implement me
	panic("implement me")
}

func (d *BridgeNetworkDriver) Delete(network *Network) error {
	//TODO implement me
	//网络名即Linux Bridge的名字
	bridgeName := network.Name
	//通过netlink库的LinkByName方法找到需要删除得网络接口
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}
	//删除网络对应的Linux Bridge设备
	return netlink.LinkDel(br)
	//后续补充删除iptables规则
}
