package main

import (
	"os"
	"log"
	"os/exec"
	"syscall"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Usage: go run main.go run /bin/bash")
	}

	//flag run or int
	flag := os.Args[1]
	command := os.Args[2]

	if flag == "run" {
		Run(command)
	}

	if flag == "init" {
		Init(command)
	}

}

func Run(command string) {
	cmd := exec.Command("/proc/self/exe", "init", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		//设置进程的namespace
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS | syscall.CLONE_NEWUSER | syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      0,
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      0,
				Size:        1,
			},
		},
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatal("cmd start err: ", err)
	}

	//设置cgroup
	group := "test-limit-cgroup"
	setCgroup(group, "100m", "512", cmd.Process.Pid)

}

//setCgroup cpu and mem
func setCgroup(group, memLimit, cpuShare string, pid int) {
	//mem limit
	if err := addSubsystemLimit(group, "memory", memLimit, pid); err != nil {
		log.Fatal("add mem limit err: ", err)
	}

	//cpu limit
	if err := addSubsystemLimit(group, "cpu", cpuShare, pid); err != nil {
		log.Fatal("add cpu limit err: ", err)
	}
}

//AddSubsystemLimit cpu or mem
func addSubsystemLimit(group, subsystem, limit string, pid int) error {
	//获取cgroup path，不存在就创建
	cgroupPath, err := getCgroupPAth(group, subsystem, true)
	//设置limit
}

func getCgroupPAth(group, subsystem string, autocreate bool) (path string, err error) {

}

func Init(command string) {

}
