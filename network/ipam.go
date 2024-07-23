package network

import "net"

type IPAM struct {
}

var ipAllocator = &IPAM{}

// Allocate 从指定的subnet网段中分配IP地址
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	return
}

// Release 从指定的subnet网段中释放掉指定IP地址
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	return nil
}
