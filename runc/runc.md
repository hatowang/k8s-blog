## 1. runc
runc是一个命令行工具，用于根据OCI规范创建linux容器

## 2. 编译
````
cd github.com/opencontainers
git clone https://github.com/opencontainers/runc
cd runc

make
sudo make install
````
编译后的二进制在/usr/local/sbin/runc

## 3. 使用
**runc是一个底层工具，主要由上层工具调用**

为了使用 runc，您必须拥有 OCI 包格式的容器。 如果您安装了 Docker，您可以使用其导出方法从现有 Docker 容器获取根文件系统。 为了使用 runc，您必须拥有 OCI 包格式的容器。 如果您安装了 Docker，您可以使用其导出方法从现有 Docker 容器获取根文件系统。
````
root@mgt01:~# mkdir /mycontainer
root@mgt01:~# cd /mycontainer/
root@mgt01:/mycontainer# mkdir rootfs
root@mgt01:/mycontainer# docker create busybox
20785efd16d7cf21cb312001649976eb6950663e31dca5673ba879ce20e3f9b4
root@mgt01:/mycontainer# mkdir rootfs
root@mgt01:/mycontainer/rootfs# docker export 20785efd16d7cf21cb312001649976eb6950663e31dca5673ba879ce20e3f9b4 | tar -C rootfs -xvf -
root@mgt01:/mycontainer# runc spec
root@mgt01:/mycontainer# ls
config.json  rootfs
root@mgt01:/mycontainer# runc run mycontainerid
```` 

修改config.json，将"terminal": true"改为terminal":false，args改为"sleep", "5"

````
root@mgt01:/mycontainer# vim config.json
root@mgt01:/mycontainer# runc create mycontainerid
root@mgt01:/mycontainer# runc start mycontainerid
root@mgt01:/mycontainer# runc list
ID              PID         STATUS      BUNDLE         CREATED                          OWNER
mycontainerid   2941795     running     /mycontainer   2021-10-28T09:37:27.616535279Z   root
root@mgt01:/mycontainer# runc delete mycontainerid
```` 