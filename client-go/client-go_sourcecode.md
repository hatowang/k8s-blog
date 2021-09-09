## client-go源码分析

##1.什么是client-go
    client-go是访问k8s的资源的客户端，即通过client-go实现对k8s集群中的资源对象进行增删改查等操作。
以下是一段使用案例。代码来自pkg/kubelet/kubelet.go
````
		kubeInformers := informers.NewSharedInformerFactoryWithOptions(kubeDeps.KubeClient, 0, informers.WithTweakListOptions(func(options *metav1.ListOptions) {
			options.FieldSelector = fields.Set{metav1.ObjectNameField: string(nodeName)}.String()
		}))
		nodeLister = kubeInformers.Core().V1().Nodes().Lister()
		nodeHasSynced = func() bool {
			return kubeInformers.Core().V1().Nodes().Informer().HasSynced()
		}
		kubeInformers.Start(wait.NeverStop)
		klog.InfoS("Attempting to sync node with API server")
````

###1. NewSharedInformerFactoryWithOptions
````
// NewSharedInformerFactoryWithOptions constructs a new instance of a SharedInformerFactory with additional options.
func NewSharedInformerFactoryWithOptions(client kubernetes.Interface, defaultResync time.Duration, options ...SharedInformerOption) SharedInformerFactory {
	factory := &sharedInformerFactory{
		client:           client,//
		namespace:        v1.NamespaceAll,
		defaultResync:    defaultResync,
		informers:        make(map[reflect.Type]cache.SharedIndexInformer),
		startedInformers: make(map[reflect.Type]bool),
		customResync:     make(map[reflect.Type]time.Duration),
	}

	// Apply all options
	for _, opt := range options {
		factory = opt(factory)
	}

	return factory
}
````

参数分析：func NewSharedInformerFactoryWithOptions(client kubernetes.Interface, defaultResync time.Duration, options ...SharedInformerOption) SharedInformerFactory 
#### 1.1 client 初始化方法为clientset.NewForConfig(clientConfig)，返回Clientset对象

ClientSet英文解释如下
````
contains the clients for groups. Each group has exactly one version included in a Clientset.
````
即ClientSet包含一组client，每一个client都代表一种k8s对象的客户端

#### 1.2 defaultResync，此处传为0

#### 1.3 第三个参数返回了 type SharedInformerOption func(*sharedInformerFactory) *sharedInformerFactory，是设置sharedInformerFactory类型的对象的tweakListOptions属性，
此处tweakListOptions的作用是将metav1.ListOptions对象的FieldSelector设置为fields.Set{metav1.ObjectNameField: string(nodeName)}.String()
````
informers.WithTweakListOptions(func(options *metav1.ListOptions) {
            //即设置FieldSelector为metadata.name=${nodeName},查询时匹配node的label
			options.FieldSelector = fields.Set{metav1.ObjectNameField: string(nodeName)}.String()
		})
````

####1.4主逻辑，创建了sharedInformerFactory的对象，设置了client、defaultResync、循环options（即第三个分析中的WithTweakListOptions），将创建了sharedInformerFactory的对象作为参数，执行WithTweakListOptions对应的方法，
即设置factory的tweakListOptions属性。

### 2. 有了创建了sharedInformerFactory的对象对象，再新建对象
````
nodeLister = kubeInformers.Core().V1().Nodes().Lister()
````

#### 2.1 Core方法
````
func (f *sharedInformerFactory) Core() core.Interface {
	return core.New(f, f.namespace, f.tweakListOptions)
}
````
Core方法中调用了New方法返回group对象，设置group对象的factory为第一步创出来的factory，设置namepace为v1.NamespaceAll,设置tweakListOptions为第一部中第四步设置的tweakListOptions.
````
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &group{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}
````

#### 2.2 V1 s
````
// V1 returns a new v1.Interface.
func (g *group) V1() v1.Interface {
	return v1.New(g.factory, g.namespace, g.tweakListOptions)
}
````
调用New方法，返回了version对象，设置了对象的一些属性factory、namespace、tweakListOptions
````
// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}
````

#### 2.3 Nodes方法，返回nodeInformer对象，设置了对象的一些属性factory、tweakListOptions
````
// Nodes returns a NodeInformer.
func (v *version) Nodes() NodeInformer {
	return &nodeInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
````

#### 2.4 Lister方法
````
func (f *nodeInformer) Lister() v1.NodeLister {
	return v1.NewNodeLister(f.Informer().GetIndexer())
}
````
参数是f.Informer().GetIndexer()，调用NewNodeLister方法，返回了v1.NodeLister对象（姑且不知道这个东西的作用）
````
// NewNodeLister returns a new NodeLister.
func NewNodeLister(indexer cache.Indexer) NodeLister {
	return &nodeLister{indexer: indexer}
}
````
NewNodeLister放法的参数是indexer，由f.Informer().GetIndexer()生成;而这个方法又调用了方法InformerFor，参数为&corev1.Node{}, f.defaultInformer
````
func (f *nodeInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&corev1.Node{}, f.defaultInformer)
}
````
InformerFor方法，创建了cache.SharedIndexInformer对象
````
// InternalInformerFor returns the SharedIndexInformer for obj using an internal
// client.
func (f *sharedInformerFactory) InformerFor(obj runtime.Object, newFunc internalinterfaces.NewInformerFunc) cache.SharedIndexInformer {
	f.lock.Lock()
	defer f.lock.Unlock()
    //获取&corev1.Node{}的反射类型informerType
	informerType := reflect.TypeOf(obj)	
	informer, exists := f.informers[informerType]
	if exists {
		return informer
	}
    //获取resyncPeriod，如果不存在设置为factory的defaultResync即为0
	resyncPeriod, exists := f.customResync[informerType]
	if !exists {
		resyncPeriod = f.defaultResync
	}
    //newFunc产生informer并存入factory的informers（map[reflect.Type]cache.SharedIndexInformer）中
    //newfunc即defaultInformer，下面分析
	informer = newFunc(f.client, resyncPeriod)
	f.informers[informerType] = informer

	return informer
}
```
//defaultInformer 入参为client和resyncPeriod，又去调用NewFilteredNodeInformer，入参为cache.Indexers{cache.NamespaceIndex:
 //cache.MetaNamespaceIndexFunc}，map有一个k即cache.NamespaceIndex（“namespace”），v即MetaNamespaceIndexFunc方法（用来获取对象的namespace）
 //以及参数tweakListOptions（前面分析过）
func (f *nodeInformer) defaultInformer(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredNodeInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

//返回cache.SharedIndexInformer对象
func NewFilteredNodeInformer(client kubernetes.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	//调用NewSharedIndexInformer方法
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				//listNode方法
				return client.CoreV1().Nodes().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
				//设置options的FieldSelector属性
					tweakListOptions(&options)
				}
				//watch方法			
				return client.CoreV1().Nodes().Watch(context.TODO(), options)
			},
		},
		&corev1.Node{},
		resyncPeriod,
		indexers,//持久化存储
	)
}
//返回SharedIndexInformer对象
func NewSharedIndexInformer(lw ListerWatcher, exampleObject runtime.Object, defaultEventHandlerResyncPeriod time.Duration, indexers Indexers) SharedIndexInformer {
	//RealClock really calls time.Now()
	realClock := &clock.RealClock{}
	sharedIndexInformer := &sharedIndexInformer{
		processor:                       &sharedProcessor{clock: realClock},
		indexer:                         NewIndexer(DeletionHandlingMetaNamespaceKeyFunc, indexers),
		listerWatcher:                   lw,//设置lw方法
		objectType:                      exampleObject, //此处即Node
		resyncCheckPeriod:               defaultEventHandlerResyncPeriod, //0
		defaultEventHandlerResyncPeriod: defaultEventHandlerResyncPeriod, //0
		//is able to monitor objects for mutation within a limited window of time
		cacheMutationDetector:           NewCacheMutationDetector(fmt.Sprintf("%T", exampleObject)),
		clock:                           realClock,
	}
	return sharedIndexInformer
}

````

###2. nodeHasSynced，delegates to the Config's Queue
````
nodeHasSynced = func() bool {
			return kubeInformers.Core().V1().Nodes().Informer().HasSynced()
		}
````

###3. 开启SharedInformerFactory对象的主逻辑
````
kubeInformers.Start(wait.NeverStop)

// Start initializes all requested informers.
func (f *sharedInformerFactory) Start(stopCh <-chan struct{}) {
	f.lock.Lock()
	defer f.lock.Unlock()
        //循环informers中的SharedIndexInformer对象，开始SharedIndexInformer的主逻辑，上面分析map中存入了Node的lw的SharedIndexInformer
	for informerType, informer := range f.informers {
		if !f.startedInformers[informerType] {
		    //开始Node的informer主逻辑
			go informer.Run(stopCh)
			//如果startedInformer开始运行了就设置为true，防止重复
			f.startedInformers[informerType] = true
		}
	}
}

func (s *sharedIndexInformer) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash() //异常处理
    //新建DeltaFIFO对象
	fifo := NewDeltaFIFOWithOptions(DeltaFIFOOptions{
		KnownObjects:          s.indexer, //本地存储
		EmitDeltaTypeReplaced: true,
	})
    
    //新建Config对象
	cfg := &Config{
		Queue:            fifo,
		ListerWatcher:    s.listerWatcher,
		ObjectType:       s.objectType,
		FullResyncPeriod: s.resyncCheckPeriod,
		RetryOnError:     false,
		ShouldResync:     s.processor.shouldResync,

		Process:           s.HandleDeltas,
		WatchErrorHandler: s.watchErrorHandler,
	}
    //匿名方法
	func() {
		s.startedLock.Lock()
		defer s.startedLock.Unlock()
        //新建controller
		s.controller = New(cfg)
		s.controller.(*controller).clock = s.clock
		s.started = true
	}()

	// Separate stop channel because Processor should be stopped strictly after controller
	processorStopCh := make(chan struct{})
	var wg wait.Group
	defer wg.Wait()              // Wait for Processor to stop
	defer close(processorStopCh) // Tell Processor to stop
	wg.StartWithChannel(processorStopCh, s.cacheMutationDetector.Run)
	wg.StartWithChannel(processorStopCh, s.processor.run)

	defer func() {
		s.startedLock.Lock()
		defer s.startedLock.Unlock()
		s.stopped = true // Don't want any new listeners
	}()
	//运行controller逻辑
	s.controller.Run(stopCh)
}

// items. See also the comment on DeltaFIFO.
func NewDeltaFIFOWithOptions(opts DeltaFIFOOptions) *DeltaFIFO {
	if opts.KeyFunction == nil {
	    //默认keyFunc name/namespace 
		opts.KeyFunction = MetaNamespaceKeyFunc
	}
    //初始化DeltaFIFO
	f := &DeltaFIFO{
		items:        map[string]Deltas{},
		queue:        []string{},
		keyFunc:      opts.KeyFunction,
		knownObjects: opts.KnownObjects, //

		emitDeltaTypeReplaced: opts.EmitDeltaTypeReplaced,
	}
	f.cond.L = &f.lock
	return f
}


// Run begins processing items, and will continue until a value is sent down stopCh or it is closed.
// It's an error to call Run more than once.
// Run blocks; call via go.
func (c *controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	go func() {
		<-stopCh
		c.config.Queue.Close()
	}()
	//新建Reflector对象
	r := NewReflector(
		c.config.ListerWatcher,//lw方法
		c.config.ObjectType, //此处试Node
		c.config.Queue,//DeltaFIFO
		c.config.FullResyncPeriod, //0
	)
	r.ShouldResync = c.config.ShouldResync
	r.WatchListPageSize = c.config.WatchListPageSize
	r.clock = c.clock
	if c.config.WatchErrorHandler != nil {
		r.watchErrorHandler = c.config.WatchErrorHandler
	}

	c.reflectorMutex.Lock()
	c.reflector = r
	c.reflectorMutex.Unlock()

	var wg wait.Group

	wg.StartWithChannel(stopCh, r.Run)
    //处理逻辑processLoop
	wait.Until(c.processLoop, time.Second, stopCh)
	wg.Wait()
}


func (c *controller) processLoop() {
	for {
	    //从DeltaFIFO中取出
		obj, err := c.config.Queue.Pop(PopProcessFunc(c.config.Process))
		if err != nil {
			if err == ErrFIFOClosed {
				return
			}
			if c.config.RetryOnError {
				// This is the safe way to re-enqueue.
				c.config.Queue.AddIfNotPresent(obj)
			}
		}
	}
}

func (f *DeltaFIFO) Pop(process PopProcessFunc) (interface{}, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	for {
	    //如果DeltaFIFO中没有元素则wait阻塞
		for len(f.queue) == 0 {
			// When the queue is empty, invocation of Pop() is blocked until new item is enqueued.
			// When Close() is called, the f.closed is set and the condition is broadcasted.
			// Which causes this loop to continue and return from the Pop().
			if f.closed {
				return nil, ErrFIFOClosed
			}

			f.cond.Wait()
		}
		//去除队列中首个元素
		id := f.queue[0]
		f.queue = f.queue[1:]
		//剩余的元素
		depth := len(f.queue)
		//第一次执行Replace()方法时插入的个数，大于0则--
		if f.initialPopulationCount > 0 {
			f.initialPopulationCount--
		}
		//根据id拿到收个Deltas
		item, ok := f.items[id]
		if !ok {
			// This should never happen
			klog.Errorf("Inconceivable! %q was in f.queue but not f.items; ignoring.", id)
			continue
		}
		delete(f.items, id)
		// Only log traces if the queue depth is greater than 10 and it takes more than
		// 100 milliseconds to process one item from the queue.
		// Queue depth never goes high because processing an item is locking the queue,
		// and new items can't be added until processing finish.
		// https://github.com/kubernetes/kubernetes/issues/103789
		if depth > 10 {
			trace := utiltrace.New("DeltaFIFO Pop Process",
				utiltrace.Field{Key: "ID", Value: id},
				utiltrace.Field{Key: "Depth", Value: depth},
				utiltrace.Field{Key: "Reason", Value: "slow event handlers blocking the queue"})
			defer trace.LogIfLong(100 * time.Millisecond)
		}
		//处理拿到的Deltas（存储的是对象及动作（addition, deletion, etc)
		err := process(item)
		if e, ok := err.(ErrRequeue); ok {
		    //如果处理失败了则重新重新放回队列
			f.addIfNotPresent(id, item)
			err = e.Err
		}
		// Don't need to copyDeltas here, because we're transferring
		// ownership to the caller.
		return item, err
	}
}

func (f *DeltaFIFO) addIfNotPresent(id string, deltas Deltas) {
	f.populated = true
	if _, exists := f.items[id]; exists {
		return
	}
    //添加进DeltaFIFO，并唤醒所有阻塞的地方（如pop方法）
	f.queue = append(f.queue, id)
	f.items[id] = deltas
	f.cond.Broadcast()
}

//分析下Process处理deltas的逻辑
    func (s *sharedIndexInformer) HandleDeltas(obj interface{}) error {
	s.blockDeltas.Lock()
	defer s.blockDeltas.Unlock()

	// from oldest to newest
	for _, d := range obj.(Deltas) {        
		switch d.Type {
		case Sync, Replaced, Added, Updated:
		    //添加到队列中
			s.cacheMutationDetector.AddObject(d.Object)
			//放入本地缓存，添加或者替换到本地缓存，调用distribute方法，分发处理
			if old, exists, err := s.indexer.Get(d.Object); err == nil && exists {
				if err := s.indexer.Update(d.Object); err != nil {
					return err
				}

				isSync := false
				switch {
				case d.Type == Sync:
					// Sync events are only propagated to listeners that requested resync
					isSync = true
				case d.Type == Replaced:
					if accessor, err := meta.Accessor(d.Object); err == nil {
						if oldAccessor, err := meta.Accessor(old); err == nil {
							// Replaced events that didn't change resourceVersion are treated as resync events
							// and only propagated to listeners that requested resync
							isSync = accessor.GetResourceVersion() == oldAccessor.GetResourceVersion()
						}
					}
				}
				s.processor.distribute(updateNotification{oldObj: old, newObj: d.Object}, isSync)
			} else {
				if err := s.indexer.Add(d.Object); err != nil {
					return err
				}
				s.processor.distribute(addNotification{newObj: d.Object}, false)
			}
		case Deleted:
		//从缓存中删掉，然后分发到listener中
			if err := s.indexer.Delete(d.Object); err != nil {
				return err
			}
			s.processor.distribute(deleteNotification{oldObj: d.Object}, false)
		}
	}
	return nil
}

// AddObject makes a deep copy of the object for later comparison.  It only works on runtime.Object
// but that covers the vast majority of our cached objects
func (d *defaultCacheMutationDetector) AddObject(obj interface{}) {
    //如果是delete则返回
	if _, ok := obj.(DeletedFinalStateUnknown); ok {
		return
	}
	//
	if obj, ok := obj.(runtime.Object); ok {
		copiedObj := obj.DeepCopyObject()

		d.addedObjsLock.Lock()
		defer d.addedObjsLock.Unlock()
		//放入addedObjs中
		d.addedObjs = append(d.addedObjs, cacheObj{cached: obj, copied: copiedObj})
	}
}

````
