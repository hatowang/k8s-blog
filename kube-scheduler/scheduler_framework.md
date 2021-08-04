## kubernetes调度框架

调度框架定义了一组扩展点，可以通过扩展扩展点来定义自定义逻辑。扩展点需要注册到调度器上，在调度时调用

### 1. 扩展点

- `QueueSort`：这些插件对调度队列中的悬决的 Pod 排序。 一次只能启用一个队列排序插件。
- `PreFilter`：这些插件用于在过滤之前预处理或检查 Pod 或集群的信息。 它们可以将 Pod 标记为不可调度。
- `Filter`：这些插件相当于调度策略中的断言（Predicates），用于过滤不能运行 Pod 的节点。 过滤器的调用顺序是可配置的。 如果没有一个节点通过所有过滤器的筛选，Pod 将会被标记为不可调度。
- `PreScore`：这是一个信息扩展点，可用于预打分工作。
- `Score`：这些插件给通过筛选阶段的节点打分。调度器会选择得分最高的节点。
- `Reserve`：这是一个信息扩展点，当资源已经预留给 Pod 时，会通知插件。 这些插件还实现了 `Unreserve` 接口，在 `Reserve` 期间或之后出现故障时调用。
- `Permit`：这些插件可以阻止或延迟 Pod 绑定。
- `PreBind`：这些插件在 Pod 绑定节点之前执行。
- `Bind`：这个插件将 Pod 与节点绑定。绑定插件是按顺序调用的，只要有一个插件完成了绑定，其余插件都会跳过。绑定插件至少需要一个。
- `PostBind`：这是一个信息扩展点，在 Pod 绑定了节点之后调用。
- `UnReserve`：这是一个信息扩展点，如果一个 Pod 在预留后被拒绝，并且被 `Permit` 插件搁置，它就会被调用。

### 2. 默认开启的部分插件

下面是默认启用的插件实现了一个或多个扩展点

`pkg/scheduler/framework/plugins/registry.go`

```
// NewInTreeRegistry builds the registry with all the in-tree plugins.
// A scheduler that runs out of tree plugins can register additional plugins
// through the WithFrameworkOutOfTreeRegistry option.
func NewInTreeRegistry() runtime.Registry {
   return runtime.Registry{
      selectorspread.Name:                        selectorspread.New,                        //SelectorSpread
      imagelocality.Name:                         imagelocality.New,                         //ImageLocality
      tainttoleration.Name:                       tainttoleration.New,                       //TaintToleration
      nodename.Name:                              nodename.New,                              //NodeName
      nodeports.Name:                             nodeports.New,                             //NodePorts
      nodepreferavoidpods.Name:                   nodepreferavoidpods.New,                   //NodePreferAvoidPods
      nodeaffinity.Name:                          nodeaffinity.New,                          //NodeAffinity
      podtopologyspread.Name:                     podtopologyspread.New,                     //PodTopologySpread
      nodeunschedulable.Name:                     nodeunschedulable.New,                     //NodeUnschedulable
      noderesources.FitName:                      noderesources.NewFit,                      //NodeResourcesFit
      noderesources.BalancedAllocationName:       noderesources.NewBalancedAllocation,       //NodeResourcesBalancedAllocation
      noderesources.MostAllocatedName:            noderesources.NewMostAllocated,            //NodeResourcesMostAllocated
      noderesources.LeastAllocatedName:           noderesources.NewLeastAllocated,           //NodeResourcesLeastAllocated
      noderesources.RequestedToCapacityRatioName: noderesources.NewRequestedToCapacityRatio, //RequestedToCapacityRatio
      volumebinding.Name:                         volumebinding.New,                         //VolumeBinding
      volumerestrictions.Name:                    volumerestrictions.New,                    //VolumeRestrictions
      volumezone.Name:                            volumezone.New,                            //VolumeZone
      nodevolumelimits.CSIName:                   nodevolumelimits.NewCSI,                   //NodeVolumeLimits
      nodevolumelimits.EBSName:                   nodevolumelimits.NewEBS,                   //EBSLimits
      nodevolumelimits.GCEPDName:                 nodevolumelimits.NewGCEPD,                 //GCEPDLimits
      nodevolumelimits.AzureDiskName:             nodevolumelimits.NewAzureDisk,             //AzureDiskLimits
      nodevolumelimits.CinderName:                nodevolumelimits.NewCinder,                //CinderLimits
      interpodaffinity.Name:                      interpodaffinity.New,                      //InterPodAffinity
      nodelabel.Name:                             nodelabel.New,                             //NodeLabel
      serviceaffinity.Name:                       serviceaffinity.New,                       //ServiceAffinity
      queuesort.Name:                             queuesort.New,                             //PrioritySort
      defaultbinder.Name:                         defaultbinder.New,                         //DefaultBinder
      defaultpreemption.Name:                     defaultpreemption.New,                     //DefaultPreemption
   }
}
```

默认调度算法列表如下:

- 1. `selectorSpread`:  SelectorSpread is a plugin that calculates selector spread priority.

     扩展点：PreScore、Score、NormalizeScore

- 2. `ImageLocality` :ImageLocality is a score plugin that favors nodes that already have requested pod container's images.

     扩展点: Score

- 3. `TaintToleration`:  TaintToleration is a plugin that checks if a pod tolerates a node's taints.

     扩展点：Filter、PreScore、Score、NormalizeScore

- 4. `NodeName`: NodeName is a plugin that checks if a pod spec node name matches the current node.

     扩展点：Filter

- 5.  `NodePorts`: NodePorts is a plugin that checks if a node has free ports for the requested pod ports.

     扩展点： PreFilter、Filter

- 6. `NodePreferAvoidPods`:  NodePreferAvoidPods is a plugin that priorities nodes according to the node annotation "scheduler.alpha.kubernetes.io/preferAvoidPods".

     即基于节点的 [注解](https://kubernetes.io/zh/docs/concepts/overview/working-with-objects/annotations/) scheduler.alpha.kubernetes.io/preferAvoidPods打分

     扩展点：Score

- 7. `NodeAffinity`: NodeAffinity is a plugin that checks if a pod node selector matches the node label.

     扩展点： Filter、Score、NormalizeScore

- 8. `PodTopologySpread`: PodTopologySpread is a plugin that ensures pod's topologySpreadConstraints is satisfied.

     扩展点： PreFilter、Filter、PreScore、Score

- 9. `NodeUnschedulable`:  NodeUnschedulable plugin filters nodes that set node.Spec.Unschedulable=true unless，the pod tolerates {key=node.kubernetes.io/unschedulable, effect:NoSchedule} taint.

     扩展点： Filter

- 10. `NodeResourcesFit`: Fit is a plugin that checks if a node has sufficient resources.

      扩展点： PreFilter、Filter

- 11. `NodeResourcesBalancedAllocation`: BalancedAllocation is a score plugin that calculates the difference between the cpu and memory fraction of capacity, and prioritizes the host based on how close the two metrics are to each other.

      即选择资源更为均衡的节点

      扩展点： Score

- 12. NodeResourcesMostAllocated: MostAllocated is a score plugin that favors nodes with high allocation based on requested resources.

      扩展点：Score

- 13. NodeResourcesLeastAllocated： LeastAllocated is a score plugin that favors nodes with fewer allocation requested resources based on requested resources.

      扩展点： Score

- 14. RequestedToCapacityRatio： 根据已分配资源的某函数设置选择节点。

      扩展点：Score

- 15. `VolumeBinding`： VolumeBinding is a plugin that binds pod volumes in scheduling.In the Filter phase, pod binding cache is created for the pod and used in Reserve and PreBind phases.

      扩展点： PreFilter、Filter、Reserve、PreBind、Score

- 16. `VolumeRestrictions`:  VolumeRestrictions is a plugin that checks volume restrictions.

      扩展点：Filter

- 17. VolumeZone:  VolumeZone is a plugin that checks volume zone.

      扩展点： Filter

- 18. `NodeVolumeLimits`: CSILimits is a plugin that checks node volume limits.

      扩展点： Filter

- 19. `EBSLimits`: 检查节点是否满足 AWS EBS 卷限制。

      扩展点：Filter

- 20. `GCEPDLimits`： 检查该节点是否满足 GCP-PD 卷限制。

      扩展点：Filter

- 21. `AzureDiskLimits`: 检查该节点是否满足 Azure 卷限制。

      扩展点：Filter

- 22. `CinderLimits`: 检查该节点是否满足 Cinder 限制。

      扩展点：Filter

- 23. `InterPodAffinity`: InterPodAffinity is a plugin that checks inter pod affinity

      扩展点： PreFilter、Filter、PreScore、Score

- 24. `NodeLabel`: NodeLabel checks whether a pod can fit based on the node labels which match a filter that it requests.

      扩展点：Filter、Score

- 25. `ServiceAffinity`:  ServiceAffinity is a plugin that checks service affinity.

      扩展点：PreFilter、Filter、Score

- 26. `PrioritySort`: PrioritySort is a plugin that implements Priority based sorting.

      扩展点：QueueSort

- 27. `DefaultBinder`: DefaultBinder binds pods to nodes using a k8s client.

      扩展点: Bind

- 28. `DefaultPreemption`: DefaultPreemption is a PostFilter plugin implements the preemption logic.

      扩展点： PostFilter

      

* 注：由上述默认调度算法，看出默认的bind算法是`DefaultBinder`，默认QueueSort算法是`PrioritySort`，默认PostFilter算法是`DefaultPreemption`.

### 3.scheduler_framework源码分析（贴了主要的

参考官方的案例[scheduler-plugins](https://github.com/hatowang/scheduler-plugins.git )（贴主干代码）



1. #### cmd/scheduler/main.go

```
func main() {
	// Register custom plugins to the scheduler framework.
	// Later they can consist of scheduler profile(s) and hence
	// used by various kinds of workloads.
	command := app.NewSchedulerCommand(
		app.WithPlugin(capacityscheduling.Name, capacityscheduling.New),
		app.WithPlugin(coscheduling.Name, coscheduling.New),
		app.WithPlugin(loadvariationriskbalancing.Name, loadvariationriskbalancing.New),
		app.WithPlugin(noderesources.AllocatableName, noderesources.NewAllocatable),
		app.WithPlugin(noderesourcetopology.Name, noderesourcetopology.New),
		app.WithPlugin(targetloadpacking.Name, targetloadpacking.New),
		// Sample plugins below.
		app.WithPlugin(crossnodepreemption.Name, crossnodepreemption.New),
		app.WithPlugin(podstate.Name, podstate.New),
		app.WithPlugin(qos.Name, qos.New),
	)
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
```

```
// WithPlugin creates an Option based on plugin name and factory. Please don't remove this function: it is used to register out-of-tree plugins,
// hence there are no references to it from the kubernetes scheduler code base.
func WithPlugin(name string, factory runtime.PluginFactory) Option {
   return func(registry runtime.Registry) error {
      return registry.Register(name, factory)
   }
}
```

```
func(registry runtime.Registry) error {
      return registry.Register(name, factory)
   }
```

```
type Registry map[string]PluginFactory
```



入口函数，调用NewSchedulerCommand，将自定义插件注册到调度程序框架。稍后它们可以由调度程序配置文件组成，各种工作负载使用。 WithPlugin方法，返回注册函数，注册函数将扩展算法注册如Registry map中。



​      

​      

#### 2.注册到调度器中(kubernetes流程)

cmd/kube-scheduler/scheduler.go->cmd/kube-scheduler/app/server.go

NewSchedulerCommand->runCommand->Setup

```
// Setup creates a completed config and a scheduler based on the command args and options
func Setup(ctx context.Context, opts *options.Options, outOfTreeRegistryOptions ...Option) (*schedulerserverconfig.CompletedConfig, *scheduler.Scheduler, error) {
   outOfTreeRegistry := make(runtime.Registry)
   for _, option := range outOfTreeRegistryOptions {
      if err := option(outOfTreeRegistry); err != nil {
         return nil, nil, err
      }
   }
}
```

 Setup方法：outOfTreeRegistry是一个runtime.Registry（map），将插件注册到outOfTreeRegistry中

```
// New returns a Scheduler
func New(client clientset.Interface,
   informerFactory informers.SharedInformerFactory,
   recorderFactory profile.RecorderFactory,
   stopCh <-chan struct{},
   opts ...Option) (*Scheduler, error) {

  registry := frameworkplugins.NewInTreeRegistry()
	if err := registry.Merge(options.frameworkOutOfTreeRegistry); err != nil {
		return nil, err
	}
}
```



​      New方法：frameworkplugins.NewInTreeRegistry()，注册默认调度方法到runtime.Registry中， registry.Merge(options.frameworkOutOfTreeRegistry)将插件中的调度方法merge到runtime.Registry中。



调度插件注册流程，首先将插件的name及初始化方法，注册到runtime.Registry(map)中，接着注册默认调度方法到runtime.Registry中，最后合并二者。



#### 3. scheduler-framewok扩展调度kube-scheduler

官方项目：[scheduler-plugins](https://github.com/hatowang/scheduler-plugins.git )
