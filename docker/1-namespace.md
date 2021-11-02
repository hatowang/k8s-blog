## linux namespace

## 用来隔离以一系列资源，以下为可隔离的namespace

    | NameSpace类型 | 系统调用参数  | 内核版本 |
    | ------------- | ------------- | -------- |
    | Mount         | CLONE_NEWNS   | 2.4.19   |
    | UTS           | CLONE_NEWUTS  | 2.6.19   |
    | IPC           | CLONE_NEWIPC  | 2.6.19   |
    | PID           | CLONE_NEWPID  | 2.6.24   |
    | NetWork       | CLONE_NEWNET  | 2.6.29   |
    | User          | CLONE_NEWUSER | 3.8      |

    三个系统调用
    
    - clone（）创建新进程
    
    - unshare（）将进程移出某个namespace
    
    - setns（）将进程加入某个namespace

  

1. UTS：隔离hostname和domainname两个系统标识

   main.go

   ```
   package main
   
   import (
   "os/exec"
   "syscall"
   "os"
   "log"
   )
   
   func main() {
   	cmd := exec.Command("sh")
   	cmd.SysProcAttr = &syscall.SysProcAttr{
   		Cloneflags: syscall.CLONE_NEWUTS,
   	}
   
   	cmd.Stdin = os.Stdin
   	cmd.Stdout = os.Stdout
   	cmd.Stderr = os.Stderr
   	if err := cmd.Run(); err != nil {
   		log.Fatal(err)
   	}
   }
   ```

   执行go run main.go，进入到一个fork的环境，使用echo $$打印当前进程的id，并读取父进程和子进程的UTS，发现所属的UTS不同，起到了隔离的作用。

   ```
   sh-4.2# echo $$
   33665
   sh-4.2# readlink /proc/33665/ns/uts
   uts:[4026532507]
   [root@mgt01 ~]# ps -ef | grep 33665
   root      33665  33661  0 23:38 pts/0    00:00:00 sh
   [root@mgt01 ~]# readlink /proc/33661/ns/uts
   uts:[4026532505]
   ```   

   修改子进程中的hostname，发现父进程不受影响

    ```
    sh-4.2# hostname
    233
    [root@mgt01 ~]# hostname
    mgt01
    ```

2. IPC：隔离System V IPC和POSIX message queues

    - 创建IPC message queue：`ipcmk -Q`
    - 查看IPC message queue：`ipcs -q`
    
    ```
    [root@mgt01 ~]# ipcmk -Q
    Message queue id: 0
    [root@mgt01 ~]# ipcs -q
    
    ------ Message Queues --------
    key        msqid      owner      perms      used-bytes   messages
    0x95e34331 0          root       644        0            0
    
    ```
    
    main.go
    
    ```
    package main
    
    import (
    "os/exec"
    "syscall"
    "os"
    "log"
    )
    
    func main() {
        cmd := exec.Command("sh")
        cmd.SysProcAttr = &syscall.SysProcAttr{
            Cloneflags: syscall.CLONE_NEWUTS,
        }
    
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        if err := cmd.Run(); err != nil {
            log.Fatal(err)
        }
    }
    ```
    
    执行go run main.go，并创建一个meesage queue，再在宿主机上查看Message Queues。发现命名空间内的meesage queue和namespace中是隔离开的。
    
    ```
    root@mgt01 src]# go run main.go
    sh-4.2# ipcs -q
    
    ------ Message Queues --------
    key        msqid      owner      perms      used-bytes   messages
    
    sh-4.2# ipcmk -Q
    Message queue id: 0
    sh-4.2# ipcs -q
    
    ------ Message Queues --------
    key        msqid      owner      perms      used-bytes   messages
    0x6a38908b 0          root       644        0            0
    [root@mgt01 ~]# ipcs -q
    
    ------ Message Queues --------
    key        msqid      owner      perms      used-bytes   messages
    0x95e34331 0          root       644        0            0
    [root@mgt01 ~]# ipcrm --queue-id=0
    [root@mgt01 ~]# ipcs -q
    
    ------ Message Queues --------
    key        msqid      owner      perms      used-bytes   messages
    
    ```

3. PID:隔离pid，进程id。

   在宿主机中进程id为1的进程是systemd进程

   ```
   [root@mgt01 ~]# ps -ef| grep systemd
   root          1      0  0 Nov01 ?        00:00:01 /usr/lib/systemd/systemd --switched-root --system --deserialize 22
   ```

   main.go

   ```
   package main
   
   import (
   "os/exec"
   "syscall"
   "os"
   "log"
   )
   
   func main() {
   	cmd := exec.Command("sh")
   	cmd.SysProcAttr = &syscall.SysProcAttr{
   		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID,
   	}
   
   	cmd.Stdin = os.Stdin
   	cmd.Stdout = os.Stdout
   	cmd.Stderr = os.Stderr
   	if err := cmd.Run(); err != nil {
   		log.Fatal(err)
   	}
   }
   ```

   执行：go run main.go，发现id为1的进程在容器中不同

   ```
   [root@mgt01 src]# go run main.go
   sh-4.2# echo $$
   1
   ```

   
4. MOUNT: 隔离各个进程看到的挂载点的视图，执行mount()和unmount()只会影响当前namespace的文件系统，高配版chroot

   main.go

   ```
   package main
   
   import (
   "os/exec"
   "syscall"
   "os"
   "log"
   )
   
   func main() {
   	cmd := exec.Command("sh")
   	cmd.SysProcAttr = &syscall.SysProcAttr{
   		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
   	}
   
   	cmd.Stdin = os.Stdin
   	cmd.Stdout = os.Stdout
   	cmd.Stderr = os.Stderr
   	if err := cmd.Run(); err != nil {
   		log.Fatal(err)
   	}
   }
   ```

   

   执行go run main.go，再执行mount -t proc proc /proc，执行ps -ef发现容器中的进程只有两个

   ```
   sh-4.2# mount -t proc proc /proc
   sh-4.2# ps -ef
   UID         PID   PPID  C STIME TTY          TIME CMD
   root          1      0  0 03:21 pts/0    00:00:00 sh
   root          8      1  0 03:22 pts/0    00:00:00 ps -ef
   ```

5. user： 用于隔离用户组合id。宿主机上的非root用户映射成容器中的root用户
    man.go
    
    ```
    package main
    
    import (
        "os/exec"
        "syscall"
        "os"
        "log"
    )
    
    func main() {
        cmd := exec.Command("sh")
        cmd.SysProcAttr = &syscall.SysProcAttr{
            Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
                syscall.CLONE_NEWUSER,
            UidMappings: []syscall.SysProcIDMap{
                {
                    ContainerID: 1234,
                    HostID:      0,
                    Size:        1,
                },
            },
            GidMappings: []syscall.SysProcIDMap{
                {
                    ContainerID: 1234,
                    HostID:      0,
                    Size:        1,
                },
            },
        }
        
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        if err := cmd.Run(); err != nil {
            log.Fatal(err)
        }
    }
    ```
    
    使用id查看容器和主机的用户信息
    
    ```
    $ id
    uid=1234 gid=1234 groups=1234
    root@mgt01:~# id
    uid=0(root) gid=0(root) groups=0(root)
    ```

6. NetWork: 隔离网络协议栈

   main.go

   ```
   package main
   
   import (
   	"os/exec"
   	"syscall"
   	"os"
   	"log"
   )
   
   func main() {
   	cmd := exec.Command("sh")
   	cmd.SysProcAttr = &syscall.SysProcAttr{
   		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
   			syscall.CLONE_NEWUSER | syscall.CLONE_NEWNET,
   		UidMappings: []syscall.SysProcIDMap{
   			{
   				ContainerID: 1234,
   				HostID:      0,
   				Size:        1,
   			},
   		},
   		GidMappings: []syscall.SysProcIDMap{
   			{
   				ContainerID: 1234,
   				HostID:      0,
   				Size:        1,
   			},
   		},
   	}
   
   	cmd.Stdin = os.Stdin
   	cmd.Stdout = os.Stdout
   	cmd.Stderr = os.Stderr
   	if err := cmd.Run(); err != nil {
   		log.Fatal(err)
   	}
   }
   ```

   

   使用ip a查看容器的网络信息

   ```
   $ ip a
   1: lo: <LOOPBACK> mtu 65536 qdisc noop state DOWN group default qlen 1000
       link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
   ```

   
