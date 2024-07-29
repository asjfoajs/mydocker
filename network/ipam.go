package network

import (
	"bytes"
	"encoding/gob"
	"mydocker/utils"
	"net"
	"os"
	"path"
)

const ipamDefaultAllocatorPath = "/var/run/mydocker/network/ipam/subnet.gob"

// 存放IP地址分配信息
type IPAM struct {
	//分配文件存放位置
	SubnetAllocatorPath string
	//网段和位图算法的数组map，key为网段，value为位图算法的数组
	Subnets *map[string]*utils.BitMap
}

// 初始化一个IPAM的对象，默认是用"/var/run/mydocker/network/ipam/subnet.json"作为分配信息存储位置
var ipAllocator = &IPAM{
	SubnetAllocatorPath: ipamDefaultAllocatorPath,
}

// load 加载网络地址分配信息
func (ipam *IPAM) load() error {
	//通过os.Stat函数检查存储文件状态，如果不存在，则说明之前没有分配，则不需要加载
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	//打开并读取存储文件
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}

	//反序列化二进制数据
	buffer := bytes.NewBuffer(subnetJson[:n])
	ipam.Subnets = decode(buffer)
	return nil
}

// 存储网段地址分配信息
func (ipam *IPAM) dump() error {
	//检查存储文件所在文件夹是否存在，如果不存在则创建，path.Split函数能够分隔目录和文件
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			//创建文件夹，os.MkdirAll相当于 mkdir -p <dir>
			err = os.MkdirAll(ipamConfigFileDir, 0644)
			if err != nil {
				return err
			}
		}
	}

	//打开存储文件，os.O_TRUNC表示如果文件存在则清空，os.O_CREATE表示如果文件不存在则创建
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}

	//直接序列化成二进制数据
	buffer := encode(*ipam.Subnets)
	_, err = subnetConfigFile.Write(buffer.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// Allocate 从指定的subnet网段中分配IP地址
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	//在网段中地址分配信息的数组
	ipam.Subnets = &map[string]*utils.BitMap{}
	//从文件中加载已分配的网段信息
	err = ipam.load()
	if err != nil {
		return
	}

	//net.IPNet.Mask.Size()函数会返回网段的子网掩码的总长度和网段前面的固定位的长度
	//比如"127.0.0.0/8"网段的子网掩码是“255.0.0.0”
	//那么net.IPNet.Mask.Size()的返回值就是前面255所对应的位数和总位数，即8和32
	_, subnet, _ = net.ParseCIDR(subnet.String())
	one, size := subnet.Mask.Size()

	//logrus.Infof("Allocate subnet: %s, size: %d, one: %d", subnet.String(), size, one)
	//fmt.Printf("Allocate subnet: %s, size: %d, one: %d", subnet.String(), size, one)
	//如果之前没有分配过这个网段，则初始化网段的分配配置
	if _, ok := (*ipam.Subnets)[subnet.String()]; !ok {
		//1 <<unit8(size -one)表示这个网段中有多少个IP地址
		//“size -one”是子网掩码后面的网络位，2^（size-one）表示这个网段中的可用IP数
		//而2^(size -one) 等价于 1 << uint(size-one)
		(*ipam.Subnets)[subnet.String()] = utils.NewBitmap(1 << uint(size-one))
	}

	//遍历网段的位图数组
	for i := 0; i < (*ipam.Subnets)[subnet.String()].Size(); i++ {
		//如果这个位图数组中第i个位是0，则说明这个IP地址没有被使用过，则将其设置为1，并返回这个IP地址
		if (*ipam.Subnets)[subnet.String()].IsClear(i) {
			(*ipam.Subnets)[subnet.String()].Set(i)
			//这里的ip地址是网段的IP地址，加上第i个位所对应的IP地址(比如192.168.0.0/16,这里就是192.168.0。0)
			ip = subnet.IP

			//通过网段的IP与上面的偏移相加计算出分配的IP地址，由于IP地址是uint的一个数组，
			//需要数组中的每一项加所需的值，比如网段是172.16.0.0/12，数组序号是65555，
			//那么在[172,16,0,0]一次加上uint8(65555 >> 24) uint8(65555 >> 16)
			//uint8(65555 >> 8)uint8(65555 >> 0)即【0，1，0，19】，那么ip地址就是172.17.0.19
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(i >> (8 * (t - 1)))
			}
			//由于此处IP是从1开始分配的，所以需要加1，最终返回的IP地址是172.17.0.20
			ip[3] += 1
			break
		}
	}
	//通过调用dump（）函数将网段地址分配信息存储到文件中
	err = ipam.dump()
	return
}

// Release 从指定的subnet网段中释放掉指定IP地址
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]*utils.BitMap{}
	//从文件中加载网段的分配信息
	err := ipam.load()

	if err != nil {
		return err
	}
	//计算ip地址在网段位图数组中的索引位置
	c := 0
	//将IP地址转换成4个字节的表示方式
	releaseIP := ipaddr.To4()
	//由于IP地址是从1开始分配的，所以需要减1
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		//与分配IP相反释放ip索引的方式ip地址的每一位相减之后分别左移将对赢得数值加到索引上
		c += int(releaseIP[t-1] << (8 * (4 - t)))
	}

	//将分配的位图数组中第c个位设置为0
	(*ipam.Subnets)[subnet.String()].Clear(c)

	//保存释放掉IP之后的网段IP分配信息
	err = ipam.dump()
	return err
}

// encode 序列化二进制数据
func encode(data interface{}) *bytes.Buffer {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil
	}
	return &buf
}

// decode 反序列化二进制数据
func decode(buff *bytes.Buffer) *map[string]*utils.BitMap {
	dec := gob.NewDecoder(buff)
	var v map[string]*utils.BitMap
	err := dec.Decode(&v)
	if err != nil {
		return nil
	}
	return &v
}
