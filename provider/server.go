package provider

import (
	"context"
	"errors"
	"github.com/LJJsde/lrpc/naming"
	"log"
	"reflect"
	"time"
)

var maxRegisterRetry int = 2

type ServerMethod interface {
	Register(string, interface{}) //error
	Run()
	Close()
	Shutdown()
}

type Option struct {
	Ip           string
	Port         int
	Hostname     string
	AppId        string
	Env          string
	NetProtocol  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

var DefaultOption = Option{
	NetProtocol:  "tcp",
	ReadTimeout:  5 * time.Second,
	WriteTimeout: 5 * time.Second,
}

type ServerStruct struct {
	listener   ListenerMethod //*ListenerMethod is error
	registry   naming.Registry
	cancelFunc context.CancelFunc
	option     Option
	Plugins    PluginContainer
}

func CreateServer(option Option, registry naming.Registry) *ServerStruct {
	if option.NetProtocol == "" {
		option.NetProtocol = DefaultOption.NetProtocol
	}
	if option.ReadTimeout == 0 {
		option.ReadTimeout = DefaultOption.ReadTimeout
	}
	if option.WriteTimeout == 0 {
		option.WriteTimeout = DefaultOption.WriteTimeout
	}
	return &ServerStruct{
		listener: CreateListener(option),
		registry: registry,
		option:   option,
		Plugins:  &pluginContainer{},
	}
}

//register service
func (svr *ServerStruct) Register(class interface{}) {
	name := reflect.Indirect(reflect.ValueOf(class)).Type().Name()
	svr.RegisterName(name, class)
}

func (svr *ServerStruct) RegisterName(name string, class interface{}) {
	handler := &HandlerStruct{class: reflect.ValueOf(class)}
	svr.listener.SetHandler(name, handler)
	svr.Plugins.RegisterHook(name, class)
	log.Printf("%s registered success!\n", name)
}

//service start
func (svr *ServerStruct) Run() {
	//先启动后暴露服务
	svr.listener.SetPlugins(svr.Plugins)
	err := svr.listener.Run()
	if err != nil {
		panic(err)
	}

	//register in discovery,注册失败（重试2次）退出服务
	err = svr.registerToNaming()
	if err != nil {
		svr.Close()
		panic(err)
	}
}

//service close
func (svr *ServerStruct) Close() {
	log.Println("close and cancel: ", svr.option.AppId, svr.option.Hostname)
	//从服务注册中心注销
	if svr.cancelFunc != nil {
		svr.cancelFunc()
	}
	//关闭当前服务
	if svr.listener != nil {
		svr.listener.Close()
	}
}

//service shutdown gracefully
func (svr *ServerStruct) Shutdown() {
	log.Println("shutdown and cancel:", svr.option.AppId, svr.option.Hostname)
	//从服务注册中心注销
	if svr.cancelFunc != nil {
		svr.cancelFunc()
	}
	//关闭当前服务
	if svr.listener != nil {
		svr.listener.Shutdown()
	}
}

func (svr *ServerStruct) registerToNaming() error {
	instance := &naming.Instance{
		Env:      svr.option.Env,
		AppId:    svr.option.AppId,
		Hostname: svr.option.Hostname,
		Addrs:    svr.listener.GetAddrs(),
	}
	retries := maxRegisterRetry
	for retries > 0 {
		retries--
		cancel, err := svr.registry.Register(context.Background(), instance)
		if err == nil {
			log.Println("register to naming server success: ", svr.option.AppId, svr.option.Hostname)
			svr.cancelFunc = cancel
			return nil
		}
	}
	return errors.New("register to naming server fail")
}
