#1.1.1Linux Veth
#创建两个namespace
sudo ip netns add ns1
sudo ip netns add ns2
#创建一对Veth
sudo ip link add veth0 type veth peer name veth1
#分别将两个Veth加入到两个namespace中
sudo ip link set veth0 netns ns1
sudo ip link set veth1 netns ns2
#去ns1的namespace中查看网络设备
sudo ip netns exec ns1 ip link
#配置每个veth的网络地址和NameSpace的路由
sudo ip netns exec ns1 ifconfig veth0 172.18.0.2/24 up
sudo ip netns exec ns2 ifconfig veth1 172.18.0.3/24 up
sudo ip netns exec ns1 route add default dev veth0
sudo ip netns exec ns2 route add default dev veth1
#通过veth 一端出去的包，另外一端能够直接收到
sudo ip netns exec ns1 ping 172.18.0.3

#1.1.2Linux Bridge
#创建Veth设备并将一端移入NameSpace
sudo ip netns add ns1
sudo ip link add veth0 type veth peer name veth1
sudo ip link veth1 setns ns1
#创建网桥
sudo brctl addbr br0
#挂载网络设备
sudo brctl addif br0 eth0
sudo brctl addif br0 veth0

#1.2Linux路由表
#启动虚拟网络设备，并设置它在Net Namespace中的IP地址
sudo ip link set veth0 up
sudo ip link set br0 up
sudo ip netns exec ns1 ifconfig veth1 172.18.0.2/24 up
#分别设置ns1网络空间的路由和宿主机上的路由
#default 代表0.0.0.0/0 即在Net namespace中所有流量都经过veth1的网络设备流出
sudo ip netns exec ns1 route add default dev veth1
#在宿主机上将172.18.0.0/24的网段请求路由到br0的网桥
sudo route add -net 172.18.0.0/24 dev br0

#1.3 linux iptables
#MASQUERADE
#打开IP转发
sudo sysctl -w net.ipv4.ip_forwarding=1
#对Namespace中发出的包添加网络地址转换
sudo iptables -t nat -A POSTROUTING -s 172.18.0.0/24 -j MASQUERADE
#DNAT
#将宿主机上的80端口的请求转发到Namespace中的IP上
sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j DNAT --to-destination 172.18.0.2
