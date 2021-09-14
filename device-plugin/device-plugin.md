## 设备插件

## 1.什么是设备插件？
k8s提供了一个[设备插件框架](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/resource-management/device-plugin.md)，旨在通过设备调度框架扩展出云服务商自己需要的设备资源。如kubevirt中扩展的kvm、tun、vhost-net等设备。这些扩展资源会通过kubelet将硬件资源注册到节点上，在pod使用的时候通过resources属性来设置所需要的最少硬件资源额度，以及该资源的限额。

## 2. 注册设备插件
````
service Registration {
	rpc Register(RegisterRequest) returns (Empty) {}
}


message RegisterRequest {
	// Version of the API the Device Plugin was built against
	string version = 1;
	// Name of the unix socket the device plugin is listening on
	// PATH = path.Join(DevicePluginPath, endpoint)
	string endpoint = 2;
	// Schedulable resource name. As of now it's expected to be a DNS Label
	string resource_name = 3;
        // Options to be communicated with Device Manager
        DevicePluginOptions options = 4;
}
`````
如代码所示，设备插件在注册的时候需提供三种信息：、
1. Version:设备插件的API Version
2. endpoint:设备插件的unix socket地址
3.resource_name:扩展设备名称

成功注册后，设备插件就像kubelet发送它所管理的设备列表。然后kubelet负责将这些资源发布到API服务器，作为kubelet节点状态的一部分。

然后用户需要请求其他类型的资源的时候，就可以在 Container 规范请求这类设备，但是有以下的限制：
- 扩展资源仅可作为整数资源使用，并且不能被过量使用
- 设备不能在容器之间共享

## 3. 设备插件的实现
设备插件常规实现步骤：
- 初始化：在这个阶段设备插件将执行特定的初始化和设置，以确保设备就绪
- 启动grpc服务，该服务监听主机路径 /var/lib/kubelet/device-plugins/ 下的 Unix 套接字，该服务会实现以下接口：
````
// DevicePlugin is the service advertised by Device Plugins
service DevicePlugin {
	// GetDevicePluginOptions returns options to be communicated with Device
        // Manager
	rpc GetDevicePluginOptions(Empty) returns (DevicePluginOptions) {}

	// ListAndWatch returns a stream of List of Devices
	// Whenever a Device state change or a Device disapears, ListAndWatch
	// returns the new list
	rpc ListAndWatch(Empty) returns (stream ListAndWatchResponse) {}

	// Allocate is called during container creation so that the Device
	// Plugin can run device specific operations and instruct Kubelet
	// of the steps to make the Device available in the container
	rpc Allocate(AllocateRequest) returns (AllocateResponse) {}

        // PreStartContainer is called, if indicated by Device Plugin during registeration phase,
        // before each container start. Device plugin can run device specific operations
        // such as reseting the device before making devices available to the container
	rpc PreStartContainer(PreStartContainerRequest) returns (PreStartContainerResponse) {}
}
````