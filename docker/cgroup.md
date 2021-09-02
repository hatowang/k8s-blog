### 深入cgroup

### 1. 概述
理解cgroup，从使用到原理

#### 1.1 cpu和cpuset
````
root@mgt01:/sys/fs/cgroup# ls
blkio  cpuacct      cpuset   freezer  memory   net_cls,net_prio  perf_event  rdma     unified
cpu    cpu,cpuacct  devices  hugetlb  net_cls  net_prio          pids        systemd

````

如上和cpu相关的有cpu、cpuset、cpuacct。cpu用于对cpu使用率的划分；cpuset用于设置cpu的亲和性等，主要用于numa架构的os；cpuacct记录了CPU的部分
信息。对CPU的资源设置可以从两个维度来考察：cpu使用百分比和CPU内核数目。前者使用cpu subsystem进行配置，后者使用cpuset subsystem进行配置。

#### 1.2 cpu subsystem
cgroup使用两种方式进行cpu调度：

- 完全公平调度（cfs）：按照比例进行调度
- 实时程序调度（rt）：用于限制实时任务，一般用不到。
### 2.cgroup使用

#### 2.1. 安装cgroup
````
yum install libcgroup libcgroup-tools  numactl  -y
````

#### 2.2 绑定进程到指定cpu核心数

获取cpu核心数
````
 [root@master1 ~]# cat /proc/cpuinfo | grep processor| sort -u| wc -l
 4
````


 获取CPU NUMA内存节点信息
````
[root@master1 ~]# numactl --hardware
available: 1 nodes (0)
node 0 cpus: 0 1 2 3
node 0 size: 4095 MB
node 0 free: 236 MB
node distances:
node   0
  0:  10
````

 

