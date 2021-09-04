# kubelet源码分析

------

本篇文章旨在通过研究源码的方式了解kubelet服务如何创建启动pod流程。

## kubelet概述

kubelet运行在k8s集群的每个节点之上，它可以通过 hostname或者覆盖了hostname的参数或者云服务提供商的特定逻辑向apiserver注册自身。

kubelet是基于PodSPec来工作的	，每个PodSpec描述了Pod的yaml或者Json对象。kubelet接受各种机制（主要是apiserver）提供的一组PodSpec，并确保PodSpec描述的容器处于running状态，并且运行状况良好。

PodSpec对象来源：

1. 文件：利用命令参数传递路径。kubelet周期性的监测此路径下的容器清单是否有更新，同步容器最新的配置。默认周期为20s，可以使用配置参数更改。
2. HTTP端点：利用命令行参数指定HTTP端点。此端点默认周期为20s，可配置。
3. HTTP服务器（HTTP server）：kubelet还可以侦听HTTP并响应简单的api来提交新的清单。



## kubeletFlags解析

```
// KubeletFlags contains configuration flags for the Kubelet.
// A configuration field should go in KubeletFlags instead of KubeletConfiguration if any of these are true:
// - its value will never, or cannot safely be changed during the lifetime of a node, or
// - its value cannot be safely shared between nodes at the same time (e.g. a hostname);
//   KubeletConfiguration is intended to be shared between nodes.
// In general, please try to avoid adding flags or configuration fields,
// we already have a confusingly large amount of them.
type KubeletFlags struct {
	//kubeconfig 配置文件的路径，指定如何连接到 API 服务器。 提供 --kubeconfig 将启用 API 服务器模式，而省略 --kubeconfig 将启用独立模式。
	KubeConfig          string
	//某 kubeconfig 文件的路径，该文件将用于获取 kubelet 的客户端证书。 如果 --kubeconfig 所指定的文件不存在，则使用引导所用 kubeconfig 从 API 服务器请求客户端证书。成功后，将引用生成的客户端证书和密钥的 kubeconfig 写入 --kubeconfig 所指定的路径。客户端证书和密钥文件将存储在 --cert-dir 所指的目录。
	BootstrapKubeconfig string

	// Crash immediately, rather than eating panics.
	//设置为 true 表示发生失效时立即崩溃。仅用于测试。 已弃用：将在未来版本中移除。
	ReallyCrashForTesting bool

	// HostnameOverride is the hostname used to identify the kubelet instead
	// of the actual hostname.
	//如果为非空，将使用此字符串而不是实际的主机名作为节点标识。如果设置了 --cloud-provider，则云驱动将确定节点的名称 （请查阅云服务商文档以确定是否以及如何使用主机名）。
	HostnameOverride string
	// NodeIP is IP address of the node.
	// If set, kubelet will use this IP address for the node.
	//节点的 IP 地址。如果设置，kubelet 将使用该 IP 地址作为节点的 IP 地址。
	NodeIP string

	// Container-runtime-specific options.
	//容器运行时相关的参数
	config.ContainerRuntimeOptions

	// certDirectory is the directory where the TLS certs are located.
	// If tlsCertFile and tlsPrivateKeyFile are provided, this flag will be ignored.
	//TLS 证书所在的目录。如果设置了 --tls-cert-file 和 --tls-private-key-file， 则此标志将被忽略。
	CertDirectory string

	// cloudProvider is the provider for cloud services.
	// +optional
	//云服务的提供者。设置为空字符串表示在没有云驱动的情况下运行。 如果设置了此标志，则云驱动负责确定节点的名称（参考云提供商文档以确定是否以及如何使用主机名）。 已弃用：将在 1.23 版本中移除，以便于从 kubelet 中去除云驱动代码。
	CloudProvider string

	// cloudConfigFile is the path to the cloud provider configuration file.
	// +optional
	//云驱动配置文件的路径。空字符串表示没有配置文件。 已弃用：将在 1.23 版本中移除，以便于从 kubelet 中去除云驱动代码。
	CloudConfigFile string

	// rootDirectory is the directory path to place kubelet files (volume
	// mounts,etc).
	//设置用于管理 kubelet 文件的根目录（例如挂载卷的相关文件等）。默认/var/lib/kubelet
	RootDirectory string

	// The Kubelet will use this directory for checkpointing downloaded configurations and tracking configuration health.
	// The Kubelet will create this directory if it does not already exist.
	// The path may be absolute or relative; relative paths are under the Kubelet's current working directory.
	// Providing this flag enables dynamic kubelet configuration.
	// To use this flag, the DynamicKubeletConfig feature gate must be enabled.
	//kubelet 使用此目录来保存所下载的配置，跟踪配置运行状况。 如果目录不存在，则 kubelet 创建该目录。此路径可以是绝对路径，也可以是相对路径。 相对路径从 kubelet 的当前工作目录计算。 设置此参数将启用动态 kubelet 配置。必须启用 DynamicKubeletConfig 特性门控之后才能设置此标志；由于此特性为 beta 阶段，对应的特性门控当前默认为 true。
	DynamicConfigDir cliflag.StringFlag

	// The Kubelet will load its initial configuration from this file.
	// The path may be absolute or relative; relative paths are under the Kubelet's current working directory.
	// Omit this flag to use the combination of built-in default configuration values and flags.
	//kubelet 将从此标志所指的文件中加载其初始配置。此路径可以是绝对路径，也可以是相对路径。 相对路径按 kubelet 的当前工作目录起计。省略此参数时 kubelet 会使用内置的默认配置值。 命令行参数会覆盖此文件中的配置。
	KubeletConfigFile string

	// registerNode enables automatic registration with the apiserver.
	//将本节点注册到 API 服务器。如果未提供 --kubeconfig 标志设置， 则此参数无关紧要，因为 kubelet 将没有要注册的 API 服务器。
	RegisterNode bool

	// registerWithTaints are an array of taints to add to a node object when
	// the kubelet registers itself. This only takes effect when registerNode
	// is true and upon the initial registration of the node.
	//设置本节点的污点标记，格式为 <key>=<value>:<effect>， 以逗号分隔。当 --register-node 为 false 时此标志无效。 已弃用：将在未来版本中移除。
	RegisterWithTaints []core.Taint

	// WindowsService should be set to true if kubelet is running as a service on Windows.
	// Its corresponding flag only gets registered in Windows builds.
	WindowsService bool

	// WindowsPriorityClass sets the priority class associated with the Kubelet process
	// Its corresponding flag only gets registered in Windows builds
	// The default priority class associated with any process in Windows is NORMAL_PRIORITY_CLASS. Keeping it as is
	// to maintain backwards compatibility.
	// Source: https://docs.microsoft.com/en-us/windows/win32/procthread/scheduling-priorities
	WindowsPriorityClass string

	// remoteRuntimeEndpoint is the endpoint of remote runtime service
	//[实验性特性] 远程运行时服务的端点。目前支持 Linux 系统上的 UNIX 套接字和 Windows 系统上的 npipe 和 TCP 端点。例如： unix:///var/run/dockershim.sock、 npipe:////./pipe/dockershim。
	RemoteRuntimeEndpoint string
	// remoteImageEndpoint is the endpoint of remote image service
	//[实验性特性] 远程镜像服务的端点。若未设定则默认情况下使用 --container-runtime-endpoint 的值。目前支持的类型包括在 Linux 系统上的 UNIX 套接字端点和 Windows 系统上的 npipe 和 TCP 端点。 例如：unix:///var/run/dockershim.sock、npipe:////./pipe/dockershim。
	RemoteImageEndpoint string
	// experimentalMounterPath is the path of mounter binary. Leave empty to use the default mount path
	//[实验性特性] 卷挂载器（mounter）的可执行文件的路径。设置为空表示使用默认挂载器 mount。 已弃用：将在 1.23 版本移除以支持 CSI。
	ExperimentalMounterPath string
	// This flag, if set, enables a check prior to mount operations to verify that the required components
	// (binaries, etc.) to mount the volume are available on the underlying node. If the check is enabled
	// and fails the mount operation fails.
	///[实验性特性] 设置为 true 表示 kubelet 在进行挂载卷操作之前要 在本节点上检查所需的组件（如可执行文件等）是否存在。 已弃用：将在 1.23 版本中移除，以便使用 CSI。
	ExperimentalCheckNodeCapabilitiesBeforeMount bool
	// This flag, if set, will avoid including `EvictionHard` limits while computing Node Allocatable.
	// Refer to [Node Allocatable](https://git.k8s.io/community/contributors/design-proposals/node/node-allocatable.md) doc for more information.
	ExperimentalNodeAllocatableIgnoreEvictionThreshold bool
	// Node Labels are the node labels to add when registering the node in the cluster
	//	<警告：alpha 特性> kubelet 在集群中注册本节点时设置的标签。标签以 key=value 的格式表示，多个标签以逗号分隔。名字空间 kubernetes.io 中的标签必须以 kubelet.kubernetes.io 或 node.kubernetes.io 为前缀， 或者在以下明确允许范围内： beta.kubernetes.io/arch, beta.kubernetes.io/instance-type, beta.kubernetes.io/os, failure-domain.beta.kubernetes.io/region, failure-domain.beta.kubernetes.io/zone, kubernetes.io/arch, kubernetes.io/hostname, kubernetes.io/os, node.kubernetes.io/instance-type, topology.kubernetes.io/region, topology.kubernetes.io/zone。
	NodeLabels map[string]string
	// lockFilePath is the path that kubelet will use to as a lock file.
	// It uses this file as a lock to synchronize with other kubelet processes
	// that may be running.
	//<警告：alpha 特性> kubelet 使用的锁文件的路径。
	LockFilePath string
	// ExitOnLockContention is a flag that signifies to the kubelet that it is running
	// in "bootstrap" mode. This requires that 'LockFilePath' has been set.
	// This will cause the kubelet to listen to inotify events on the lock file,
	// releasing it and exiting when another process tries to open that file.
	//设置为 true 表示当发生锁文件竞争时 kubelet 可以退出。
	ExitOnLockContention bool
	// DEPRECATED FLAGS
	// minimumGCAge is the minimum age for a finished container before it is
	// garbage collected.
	//已结束的容器在被垃圾回收清理之前的最少存活时间。 例如：300ms、10s 或者 2h45m。 已弃用：请改用 --eviction-hard 或者 --eviction-soft。 此标志将在未来的版本中删除。
	MinimumGCAge metav1.Duration
	// maxPerPodContainerCount is the maximum number of old instances to
	// retain per container. Each container takes up some disk space.
	//每个已停止容器可以保留的的最大实例数量。每个容器占用一些磁盘空间。 已弃用：应在 --config 所给的配置文件中进行设置。 
	MaxPerPodContainerCount int32
	// maxContainerCount is the maximum number of old instances of containers
	// to retain globally. Each container takes up some disk space.
	//设置全局可保留的已停止容器实例个数上限。 每个实例会占用一些磁盘空间。要禁用，请设置为负数。 已弃用：应在 --config 所给的配置文件中进行设置。
	MaxContainerCount int32
	// masterServiceNamespace is The namespace from which the kubernetes
	// master services should be injected into pods.
	MasterServiceNamespace string
	// registerSchedulable tells the kubelet to register the node as
	// schedulable. Won't have any effect if register-node is false.
	// DEPRECATED: use registerWithTaints instead
	//注册本节点为可调度的节点。当 --register-node标志为 false 时此设置无效。 已弃用：此参数将在未来的版本中删除。
	RegisterSchedulable bool
	// nonMasqueradeCIDR configures masquerading: traffic to IPs outside this range will use IP masquerade.
	//kubelet 向该 IP 段之外的 IP 地址发送的流量将使用 IP 伪装技术。 设置为 0.0.0.0/0 则不使用伪装。 已弃用：应在 --config 所给的配置文件中进行设置。
	NonMasqueradeCIDR string
	// This flag, if set, instructs the kubelet to keep volumes from terminated pods mounted to the node.
	// This can be useful for debugging volume related issues.
	//设置为 true 表示 Pod 终止后仍然保留之前挂载过的卷，常用于调试与卷有关的问题。 已弃用：将未来版本中移除。
	KeepTerminatedPodVolumes bool
	// SeccompDefault enables the use of `RuntimeDefault` as the default seccomp profile for all workloads on the node.
	// To use this flag, the corresponding SeccompDefault feature gate must be enabled.
	SeccompDefault bool
}
```

## 容器运行时相关ContainerRuntimeOptions

```
// ContainerRuntimeOptions defines options for the container runtime.
type ContainerRuntimeOptions struct {
	// General Options.

	// ContainerRuntime is the container runtime to use.
	//要使用的容器运行时。目前支持 docker、remote。默认值为docker
	ContainerRuntime string
	// RuntimeCgroups that container runtime is expected to be isolated in.
	//设置用于创建和运行容器运行时的 cgroup 的绝对名称。
	RuntimeCgroups string

	// Docker-specific options.

	// DockershimRootDirectory is the path to the dockershim root directory. Defaults to
	// /var/lib/dockershim if unset. Exposed for integration testing (e.g. in OpenShift).
	DockershimRootDirectory string
	// PodSandboxImage is the image whose network/ipc namespaces
	// containers in each pod will use.
	//所指定的镜像不会被镜像垃圾收集器删除。 当容器运行环境设置为 docker 时，各个 Pod 中的所有容器都会 使用此镜像中的网络和 IPC 名字空间。 其他 CRI 实现有自己的配置来设置此镜像。
	PodSandboxImage string
	// DockerEndpoint is the path to the docker endpoint to communicate with.
	//使用这里的端点与 docker 端点通信。 仅当容器运行环境设置为 docker 时，此特定于 docker 的参数才有效。
	DockerEndpoint string
	// If no pulling progress is made before the deadline imagePullProgressDeadline,
	// the image pulling will be cancelled. Defaults to 1m0s.
	// +optional
	//如果在该参数值所设置的期限之前没有拉取镜像的进展，镜像拉取操作将被取消。 仅当容器运行环境设置为 docker 时，此特定于 docker 的参数才有效。
	ImagePullProgressDeadline metav1.Duration

	// Network plugin options.

	// networkPluginName is the name of the network plugin to be invoked for
	// various events in kubelet/pod lifecycle
	//设置 kubelet/Pod 生命周期中各种事件调用的网络插件的名称。 仅当容器运行环境设置为 docker 时，此特定于 docker 的参数才有效。
	NetworkPluginName string
	// NetworkPluginMTU is the MTU to be passed to the network plugin,
	// and overrides the default MTU for cases where it cannot be automatically
	// computed (such as IPSEC).
	//传递给网络插件的 MTU 值，将覆盖默认值。 设置为 0 则使用默认的 MTU 1460。仅当容器运行环境设置为 docker 时， 此特定于 docker 的参数才有效。
	NetworkPluginMTU int32
	// CNIConfDir is the full path of the directory in which to search for
	// CNI config files
	//此值为某目录的全路径名。kubelet 将在其中搜索 CNI 配置文件。 仅当容器运行环境设置为 docker 时，此特定于 docker 的参数才有效。
	CNIConfDir string
	// CNIBinDir is the full path of the directory in which to search for
	// CNI plugin binaries
	// 此值为以逗号分隔的完整路径列表。 kubelet 将在所指定路径中搜索 CNI 插件的可执行文件。 仅当容器运行环境设置为 docker 时，此特定于 docker 的参数才有效。
	CNIBinDir string
	// CNICacheDir is the full path of the directory in which CNI should store
	// cache files
	//此值为一个目录的全路径名。CNI 将在其中缓存文件。 仅当容器运行环境设置为 docker 时，此特定于 docker 的参数才有效。
	CNICacheDir string

	// Image credential provider plugin options

	// ImageCredentialProviderConfigFile is the path to the credential provider plugin config file.
	// This config file is a specification for what credential providers are enabled and invokved
	// by the kubelet. The plugin config should contain information about what plugin binary
	// to execute and what container images the plugin should be called for.
	// +optional
	//指向凭据提供插件配置文件所在目录的路径。
	ImageCredentialProviderConfigFile string
	// ImageCredentialProviderBinDir is the path to the directory where credential provider plugin
	// binaries exist. The name of each plugin binary is expected to match the name of the plugin
	// specified in imageCredentialProviderConfigFile.
	// +optional
	//指向凭据提供组件可执行文件所在目录的路径。
	ImageCredentialProviderBinDir string
}
```

