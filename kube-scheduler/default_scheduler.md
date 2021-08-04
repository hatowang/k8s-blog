## kube-scheduler默认调度算法分析

代码版本：v1.20.5



**默认的调度策略**

默认调度算法：pkg/scheduler/algorithmprovider/registry.go

```
func getDefaultConfig() *schedulerapi.Plugins
```

#### 1. QueueSort

算法：PrioritySort









### 1.预选

#### 1.1 PodFitsHostPorts：

官方解释

```
Checks if a Node has free ports (the network protocol kind) for the Pod ports the Pod is requesting.
```



检查节点是否具有用于 Pod 请求的 Pod 端口的空闲端口（网络协议类型）。 



代码位置：pkg/scheduler/framework/plugins/nodeports/node_ports.go

源码分析：
第一步: PreFilter方法，获取v1.ContainerPort数组，记录待调度pod的ContainerPort

```
func getContainerPorts(pods ...*v1.Pod) []*v1.ContainerPort 
```

第二步： Filter方法，遍历ContainerPort，判断type HostPortInfo map[string]map[ProtocolPort]struct{}中是否已存在改port

```
func fitsPorts(wantPorts []*v1.ContainerPort, nodeInfo *framework.NodeInfo) bool
```

```
func (h HostPortInfo) CheckConflict(ip, protocol string, port int32) bool
```



