package network

import (
	"net"
	"testing"
)

func TestAllocate(t *testing.T) {
	//在192.168.0.0/24网段下分配IP
	_, ipnet, _ := net.ParseCIDR("192.168.0.0/24")
	ip, _ := ipAllocator.Allocate(ipnet)
	t.Log(ip)
}

func TestRelease(t *testing.T) {
	//在192.168.0.0/24网段下释放刚分配的IP 192.168.0.1
	ip, ipnet, _ := net.ParseCIDR("192.168.0.1/24")
	ipAllocator.Release(ipnet, &ip)
}
