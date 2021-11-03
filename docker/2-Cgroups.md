## Cgroups

Linux CGroup全称Linux Control Group， 是Linux内核的一个功能，用来限制，控制与分离一个进程组群的资源（如CPU、内存、磁盘输入输出等）。

Linux CGroupCgroup 可让您为系统中所运行任务（进程）的用户定义组群分配资源 — 比如 CPU 时间、系统内存、网络带宽或者这些资源的组合。您可以监控您配置的 cgroup，拒绝 cgroup 访问某些资源，甚至在运行的系统中动态配置您的 cgroup。

主要提供了如下功能：

- Resource limitation: 限制资源使用，比如内存使用上限以及文件系统的缓存限制。
- Prioritization: 优先级控制，比如：CPU利用和磁盘IO吞吐。
- Accounting: 一些审计或一些统计，主要目的是为了计费。
- Control: 挂起进程，恢复执行进程。

## cgroup 层级关系：

在 cgroups 中，任务就是系统的一个进程。

- 控制族群（control group）。控制族群就是一组按照某种标准划分的进程。Cgroups 中的资源控制都是以控制族群为单位实现。一个进程可以加入到某个控制族群，也从一个进程组迁移到另一个控制族群。一个进程组的进程可以使用 cgroups 以控制族群为单位分配的资源，同时受到 cgroups 以控制族群为单位设定的限制。

- 层级（hierarchy）。控制族群可以组织成 hierarchical 的形式，既一颗控制族群树。控制族群树上的子节点控制族群是父节点控制族群的孩子，继承父控制族群的特定的属性。

- 子系统（subsytem）。一个子系统就是一个资源控制器，比如 cpu 子系统就是控制 cpu 时间分配的一个控制器。子系统必须附加（attach）到一个层级上才能起作用，一个子系统附加到某个层级以后，这个层级上的所有控制族群都受到这个子系统的控制。

  - 在根目录执行 `lssubsys -a`查看子系统

    ```
    [root@mgt01 src]# lssubsys -a
    cpuset
    cpu,cpuacct
    memory
    devices
    freezer
    net_cls,net_prio
    blkio
    perf_event
    hugetlb
    pids
    ```

    

### 子系统的介绍

- blkio – 这个子系统为块设备设定输入/输出限制，比如物理设备（磁盘，固态硬盘，USB 等等）。
- cpu – 这个子系统使用调度程序提供对 CPU 的 cgroup 任务访问。
- cpuacct – 这个子系统自动生成 cgroup 中任务所使用的 CPU 报告。
- cpuset – 这个子系统为 cgroup 中的任务分配独立 CPU（在多核系统）和内存节点。
- devices – 这个子系统可允许或者拒绝 cgroup 中的任务访问设备。
- freezer – 这个子系统挂起或者恢复 cgroup 中的任务。
- memory – 这个子系统设定 cgroup 中任务使用的内存限制，并自动生成由那些任务使用的内存资源报
- net_cls – 这个子系统使用等级识别符（classid）标记网络数据包，可允许 Linux 流量控制程序（tc）识别从具体 cgroup 中生成的数据包。

### 命令行实操

````
[root@mgt01 docker]# mkdir cgroup-test
[root@mgt01 docker]# mount -t cgroup -o none,name=cgroup-test cgroup-test ./cgroup-test
[root@mgt01 docker]# ls -al  cgroup-test/
total 0
drwxr-xr-x. 2 root root  0 Nov  2 05:18 .
drwxr-xr-x. 4 root root 65 Nov  2 05:18 ..
-rw-r--r--. 1 root root  0 Nov  2 05:18 cgroup.clone_children
--w--w--w-. 1 root root  0 Nov  2 05:18 cgroup.event_control
-rw-r--r--. 1 root root  0 Nov  2 05:18 cgroup.procs
-r--r--r--. 1 root root  0 Nov  2 05:18 cgroup.sane_behavior
-rw-r--r--. 1 root root  0 Nov  2 05:18 notify_on_release
-rw-r--r--. 1 root root  0 Nov  2 05:18 release_agent
-rw-r--r--. 1 root root  0 Nov  2 05:18 tasks
````

上述文件分析：

- cgroup.clone_children： 内容为1时子进程才会继承父进程的cpuset

- cgroup.event_control：监视状态变化和分组删除事件的配置文件
- cgroup.procs：属于分组的进程 PID 列表。仅包括多线程进程的线程 leader 的 TID，这点与 tasks 不同
- tasks： 标识该cgroup下的进程ID

在此文件中执行下边命令，创建两个子cgroup

```
mkdir cgroup-1
mkdir cgroup-2
```

将当前bash加入到cgroup中

```
[root@mgt01 cgroup-test]# sh -c "echo $$ >> tasks"
[root@mgt01 cgroup-test]# echo $$
32910
[root@mgt01 cgroup-test]# cat /proc/32910/cgroup
12:name=cgroup-test:/
11:blkio:/
10:memory:/
9:perf_event:/
8:hugetlb:/
7:pids:/
6:devices:/user.slice
5:cpuset:/
4:net_prio,net_cls:/
3:cpuacct,cpu:/
2:freezer:/
1:name=systemd:/user.slice/user-0.slice/session-2.scope
```



###  限制内存

执行命令

```
root@mgt01:~# mount |grep memory
cgroup on /sys/fs/cgroup/memory type cgroup (rw,nosuid,nodev,noexec,relatime,memory)

```

`/sys/fs/cgroup/memory`目录挂载在`memory subsystem`的 `hierarchy` 上

在`memory`下创建一个子`cgroup`，并加入内存限制

```
root@mgt01:/sys/fs/cgroup/memory/limit-memory# echo "50m" memory.limit_in_bytes
50m memory.limit_in_bytes
root@mgt01:/sys/fs/cgroup/memory/limit-memory# echo $$ > tasks
root@mgt01:/sys/fs/cgroup/memory/limit-memory# echo 1 >> memory.oom_control
root@mgt01:/sys/fs/cgroup/memory/limit-memory# stress --vm-bytes 200m --vm-keep -m 1
stress: info: [604571] dispatching hogs: 0 cpu, 0 io, 1 vm, 0 hdd
```

删除测试的`cgroup`

```
root@mgt01:/sys/fs/cgroup/memory# cgdelete memory:limit-memory
```

使用go代码限制内存

