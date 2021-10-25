## cni源码阅读(以cni plugins项目的plugins/main/bridge为例)，版本v0.8.7
[cni](https://github.com/containernetworking/cni.git)
[plugins](https://github.com/containernetworking/plugins.git)
[配置](cni/1-1 cniIntroduction.md)
###1. 入口（plugins/main/bridge/bridge.go:main）
````
func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, bv.BuildString("bridge"))
}
````
**cmdAdd, cmdCheck, cmdDel都是func(_ *CmdArgs) error的类型**
定义了添加网络、删除网络、检查网络、版本等接口

#### 2. cni怎么去调上述接口
##### 2.1 PluginMain方法，去调用PluginMainWithError方法
````
func PluginMain(cmdAdd, cmdCheck, cmdDel func(_ *CmdArgs) error, versionInfo version.PluginInfo, about string) {
	if e := PluginMainWithError(cmdAdd, cmdCheck, cmdDel, versionInfo, about); e != nil {
		if err := e.Print(); err != nil {
			log.Print("Error writing error JSON to stdout: ", err)
		}
		os.Exit(1)
	}
}
````
##### 2.2 新建dispatcher对象，调用pluginMain方法
````
func PluginMainWithError(cmdAdd, cmdCheck, cmdDel func(_ *CmdArgs) error, versionInfo version.PluginInfo, about string) *types.Error {
	return (&dispatcher{
		Getenv: os.Getenv,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}).pluginMain(cmdAdd, cmdCheck, cmdDel, versionInfo, about)
}
````

##### 2.3 读取环境变量，解析出命令和参数，
`````
func (t *dispatcher) pluginMain(cmdAdd, cmdCheck, cmdDel func(_ *CmdArgs) error, versionInfo version.PluginInfo, about string) *types.Error {
	//1. 读取环境变量CNI_COMMAND、CNI_CONTAINERID、CNI_NETNS、CNI_IFNAME、CNI_ARGS、CNI_PATH，返回cmd及初始化CmdArgs
	cmd, cmdArgs, err := t.getCmdArgsFromEnv()
	if err != nil {
		// Print the about string to stderr when no command is set
		if err.Code == types.ErrInvalidEnvironmentVariables && t.Getenv("CNI_COMMAND") == "" && about != "" {
			_, _ = fmt.Fprintln(t.Stderr, about)
			return nil
		}
		return err
	}
	//2.校验cmdArgs
	if cmd != "VERSION" {
		if err = validateConfig(cmdArgs.StdinData); err != nil {
			return err
		}
		if err = utils.ValidateContainerID(cmdArgs.ContainerID); err != nil {
			return err
		}
		if err = utils.ValidateInterfaceName(cmdArgs.IfName); err != nil {
			return err
		}
	}
	//3. 根据命令，调用checkVersionAndCall方法去调用具体的接口
	switch cmd {
	case "ADD":
		err = t.checkVersionAndCall(cmdArgs, versionInfo, cmdAdd)
	case "CHECK":
		configVersion, err := t.ConfVersionDecoder.Decode(cmdArgs.StdinData)
		if err != nil {
			return types.NewError(types.ErrDecodingFailure, err.Error(), "")
		}
		if gtet, err := version.GreaterThanOrEqualTo(configVersion, "0.4.0"); err != nil {
			return types.NewError(types.ErrDecodingFailure, err.Error(), "")
		} else if !gtet {
			return types.NewError(types.ErrIncompatibleCNIVersion, "config version does not allow CHECK", "")
		}
		for _, pluginVersion := range versionInfo.SupportedVersions() {
			gtet, err := version.GreaterThanOrEqualTo(pluginVersion, configVersion)
			if err != nil {
				return types.NewError(types.ErrDecodingFailure, err.Error(), "")
			} else if gtet {
				if err := t.checkVersionAndCall(cmdArgs, versionInfo, cmdCheck); err != nil {
					return err
				}
				return nil
			}
		}
		return types.NewError(types.ErrIncompatibleCNIVersion, "plugin version does not allow CHECK", "")
	case "DEL":
		err = t.checkVersionAndCall(cmdArgs, versionInfo, cmdDel)
	case "VERSION":
		if err := versionInfo.Encode(t.Stdout); err != nil {
			return types.NewError(types.ErrIOFailure, err.Error(), "")
		}
	default:
		return types.NewError(types.ErrInvalidEnvironmentVariables, fmt.Sprintf("unknown CNI_COMMAND: %v", cmd), "")
	}

	if err != nil {
		return err
	}
	return nil
}
`````

##### 2.4 checkVersionAndCall方法，校验cniversion，调用具体接口的方法
```
func (t *dispatcher) checkVersionAndCall(cmdArgs *CmdArgs, pluginVersionInfo version.PluginInfo, toCall func(*CmdArgs) error) *types.Error {
	//获取版本
	configVersion, err := t.ConfVersionDecoder.Decode(cmdArgs.StdinData)
	if err != nil {
		return types.NewError(types.ErrDecodingFailure, err.Error(), "")
	}
	//校验版本
	verErr := t.VersionReconciler.Check(configVersion, pluginVersionInfo)
	if verErr != nil {
		return types.NewError(types.ErrIncompatibleCNIVersion, "incompatible CNI versions", verErr.Details())
	}
	//调用具体接口方法cmdAdd、cmdCheck、cmdDel
	if err = toCall(cmdArgs); err != nil {
		if e, ok := err.(*types.Error); ok {
			// don't wrap Error in Error
			return e
		}
		return types.NewError(types.ErrInternal, err.Error(), "")
	}

	return nil
}
```

##### 2.5 CmdArgs解析
````
type CmdArgs struct {
	ContainerID string //nsid
	Netns       string//ns path
	IfName      string//网卡名称
	Args        string//CNI_ARGS
	Path        string//CNI_PATH，二进制地址
	StdinData   []byte//入参
}
````


#### 3. 创建网络（cmdAdd）
1. loadNetConf：读取配置文件
2. setupBridge：创建网桥
3. setupVeth：创建vethpair
4. 分配ip，路由等
5. 检测网卡状态


