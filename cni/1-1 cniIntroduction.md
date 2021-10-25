### 1. CNI
CNI（容器网络接口）：是CNCF项目。提供网络插件的接口编写规范和库。CNI用于容器创建时网络的创建及容器删除时网络的删除。

### 2. cni使用

1. 创建配置文件
````
$ mkdir -p /etc/cni/net.d
$ cat >/etc/cni/net.d/10-mynet.conf <<EOF
{
	"cniVersion": "0.2.0",
	"name": "mynet",
	"type": "bridge",
	"bridge": "cni0",
	"isGateway": true,
	"ipMasq": true,
	"ipam": {
		"type": "host-local",
		"subnet": "10.22.0.0/16",
		"routes": [
			{ "dst": "0.0.0.0/0" }
		]
	}
}
EOF
$ cat >/etc/cni/net.d/99-loopback.conf <<EOF
{
	"cniVersion": "0.2.0",
	"name": "lo",
	"type": "loopback"
}
EOF
````


2. 编译并测试plugin插件
````
$ cd $GOPATH/src/github.com/containernetworking/plugins
$ ./build_linux.sh # or build_windows.sh
$ ls bin
ls bin/
bandwidth  dhcp      flannel      host-local  loopback  portmap  sbr     tuning
bridge     firewall  host-device  ipvlan      macvlan   ptp      static  vlan
$ CNI_PATH=$GOPATH/src/github.com/containernetworking/plugins/bin
$ cd $GOPATH/src/github.com/containernetworking/cni/scripts
$ sudo CNI_PATH=$CNI_PATH ./priv-net-run.sh ifconfig
eth0: flags=4163<UP,BROADCAST,RUNNING,MULTICAST>  mtu 1500
        inet 10.22.0.7  netmask 255.255.0.0  broadcast 10.22.255.255
        inet6 fe80::8c11:f1ff:fef9:b37c  prefixlen 64  scopeid 0x20<link>
        ether 8e:11:f1:f9:b3:7c  txqueuelen 0  (Ethernet)
        RX packets 10  bytes 1353 (1.3 KB)
        RX errors 0  dropped 3  overruns 0  frame 0
        TX packets 11  bytes 838 (838.0 B)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0

lo: flags=73<UP,LOOPBACK,RUNNING>  mtu 65536
        inet 127.0.0.1  netmask 255.0.0.0
        inet6 ::1  prefixlen 128  scopeid 0x10<host>
        loop  txqueuelen 1000  (Local Loopback)
        RX packets 0  bytes 0 (0.0 B)
        RX errors 0  dropped 0  overruns 0  frame 0
        TX packets 0  bytes 0 (0.0 B)
        TX errors 0  dropped 0 overruns 0  carrier 0  collisions 0
````





