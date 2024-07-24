package network

import (
	"net"
	"testing"
)

func TestBridgerInit(t *testing.T) {
	d := BridgeNetworkDriver{}
	_, err := d.Create("192.168.1.0/24", "testbridge")
	t.Logf("err:%v", err)
}
func TestBridgerDelete(t *testing.T) {
	d := BridgeNetworkDriver{}
	_, ipRange, _ := net.ParseCIDR("192.168.1.0/24")

	n := &Network{
		IpRange: ipRange,
		Name:    "testbridge",
	}
	err := d.Delete(n)
	t.Logf("err:%v", err)
}

//func TestBridgeConnect(t *testing.T) {
//	ep := Endpoint{
//		ID: "testcontainer",
//	}
//
//	n := Network{
//		Name: "testbridge",
//	}
//
//	d := BridgeNetworkDriver{}
//	err := d.Connect(&n, &ep)
//	t.Logf("err:%v", err)
//}
//
//func TestNetworkConnect(t *testing.T) {
//	cInfo := &container.ContainerInfo{
//		ID:  "testcontainer",
//		Pid: "15438",
//	}
//
//	d := BridgeNetworkDriver{}
//	n, err := d.Create("192.168.1.0/24", "testbridge")
//	t.Logf("err:%v", err)
//
//	Init()
//
//	networks[n.Name] = n
//	err = Content(n.Name, cInfo)
//	t.Logf("err:%v", err)
//
//}
//
//func TestLoad(t *testing.T) {
//	n := Network{
//		Name: "testbridge",
//	}
//
//	n.load("/var/run/mydocker/network/network/testbridge")
//
//	t.Logf("network:%v", n)
//}
