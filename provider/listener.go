package provider

import (
	"errors"
	"fmt"
	"github.com/LJJsde/lrpc/config"
	"github.com/LJJsde/lrpc/protocol"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"
	"github.com/LJJsde/lrpc/codec"
)

var ServerClosedErr = errors.New("server closed error!")

type ListenerMethod interface {
	Run() error
	SetHandler(string, HandlerMethod)
	SetPlugins(PluginContainer)
	Close()
	GetAddrs() []string
	Shutdown()
}

//base on tcp
type ListenerStruct struct {
	ServiceIp   string
	ServicePort int
	option      Option
	Plugins     PluginContainer
	Handlers    map[string]HandlerMethod
	nl          net.Listener
	doneChan    chan struct{} //外层控制结束通道
	handlingNum int32         //处理中任务数
	shutdown    int32         //关闭处理中标志位
}

func CreateListener(option Option) *ListenerStruct {
	return &ListenerStruct{ServiceIp: option.Ip,
		ServicePort: option.Port,
		option:      option,
		Handlers:    make(map[string]HandlerMethod),
		doneChan:    make(chan struct{}),
	}
}

func (l *ListenerStruct) SetPlugins(plugins PluginContainer) {
	l.Plugins = plugins
}

func (l *ListenerStruct) SetHandler(name string, handler HandlerMethod) {
	if _, ok := l.Handlers[name]; ok {
		log.Printf("%s is registered!\n", name)
		return
	}
	l.Handlers[name] = handler
}

//start listening and waiting for connection
func (l *ListenerStruct) Run() error {
	//listen on port by tcp
	addr := fmt.Sprintf("%s:%d", l.ServiceIp, l.ServicePort)
	log.Println(l.option.NetProtocol, addr)
	nl, err := net.Listen(l.option.NetProtocol, addr)
	if err != nil {
		//panic(err)
		return err
	}
	l.nl = nl
	log.Printf("listen on %s success!", addr)

	//accept conn
	go l.acceptConn()
	return nil
}

func (l *ListenerStruct) acceptConn() {
	for {
		conn, err := l.nl.Accept()
		if err != nil {
			select { //done
			case <-l.getDoneChan():
				log.Println("server closed done")
				return
			default:
			}

			if e, ok := err.(net.Error); ok && e.Temporary() { //网络发生临时错误,不退出重试
				log.Printf("server accept network error: %v", err)
				time.Sleep(5 * time.Millisecond)
				continue
			}

			log.Printf("server accept err: %v\n", err)
			return
		}

		//plugin aop
		conn, ok := l.Plugins.ConnAcceptHook(conn)
		if !ok {
			conn.Close()
			continue
		}
		log.Printf("server accepted conn: %v\n", conn.RemoteAddr().String())

		//create new routine worker each connection
		go l.handleConn(conn)
	}
}

//handle each connection
func (l *ListenerStruct) handleConn(conn net.Conn) {
	//关闭挡板
	if l.isShutdown() {
		return
	}

	//catch panic
	defer func() {
		if err := recover(); err != nil {
			log.Printf("server %s catch panic err:%s\n", conn.RemoteAddr(), err)
		}
		l.CloseConn(conn)
	}()

	for {
		//关闭挡板
		if l.isShutdown() {
			return
		}

		//readtimeout
		startTime := time.Now()
		if l.option.ReadTimeout != 0 {
			conn.SetReadDeadline(startTime.Add(l.option.ReadTimeout))
		}

		//处理中任务数+1
		atomic.AddInt32(&l.handlingNum, 1)
		//任意退出都会导致处理中任务数-1
		defer atomic.AddInt32(&l.handlingNum, -1)

		//read from network
		msg, err := l.receiveData(conn)
		if err != nil || msg == nil {
			log.Println("server receive error:", err) //timeout
			return
		}

		//decode
		coder := codec.Codecs[msg.Header.SerializeType()] //get from cache
		if coder == nil {
			return
		}
		inArgs := make([]interface{}, 0)
		err = coder.Decode(msg.Payload, &inArgs) //rpcdata
		if err != nil {
			log.Println("server request decode err:%v\n", err)
			return
		}
		//log.Printf("server request decode data finish:%v\n", inArgs)

		//call local service
		handler, ok := l.Handlers[msg.ServiceClass]
		if !ok {
			log.Println("server can not found handler error:", msg.ServiceClass)
			return
		}

		l.Plugins.BeforeCallHook(msg.ServiceClass, msg.ServiceMethod, inArgs) //ctx

		result, err := handler.Handle(msg.ServiceMethod, inArgs)

		l.Plugins.AfterCallHook(msg.ServiceClass, msg.ServiceMethod, inArgs, result, err)
		//log.Println("server call local service finish! result:", result)

		//encode
		encodeRes, err := coder.Encode(result) //[]byte result + err
		if err != nil {
			log.Printf("server response encode err:%v\n", err)
			return
		}

		//send result timeout
		if l.option.WriteTimeout != 0 {
			conn.SetWriteDeadline(startTime.Add(l.option.WriteTimeout))
		}

		l.Plugins.BeforeWriteHook(encodeRes)
		err = l.sendData(conn, encodeRes)
		l.Plugins.AfterWriteHook(encodeRes, err)
		if err != nil {
			log.Printf("server send err:%v\n", err) //timeout
			return
		}

		log.Printf("server send result finish! total runtime: %v", time.Now().Sub(startTime).Seconds())
		return
	}
}

func (l *ListenerStruct) receiveData(conn net.Conn) (*protocol.MessageStruct, error) {
	l.Plugins.BeforeReadHook() //ctx

	msg, err := protocol.Read(conn)
	if err == io.EOF { //close
		log.Printf("server read finish:%v\n", err)
		return msg, nil
	}

	l.Plugins.AfterReadHook(msg, err)

	if err != nil {
		//rate limit
		return nil, err
	}
	return msg, nil
}

func (l *ListenerStruct) sendData(conn net.Conn, payload []byte) error {
	resMsg := protocol.NewMessage()
	resMsg.SetVersion(config.Protocol_MsgVersion)
	resMsg.SetMsgType(protocol.Response)
	resMsg.SetCompressType(protocol.None)
	resMsg.SetSerializeType(protocol.Gob)
	resMsg.Payload = payload
	return resMsg.Send(conn)
}

//net addr
func (l *ListenerStruct) GetAddrs() []string {
	//l.nl.Addr()
	addr := fmt.Sprintf("tcp://%s:%d", l.ServiceIp, l.ServicePort)
	return []string{addr}
}

func (l *ListenerStruct) getDoneChan() <-chan struct{} {
	return l.doneChan
}

func (l *ListenerStruct) closeDoneChan() {
	select {
	case <-l.doneChan:
	default:
		close(l.doneChan)
	}
}

func (l *ListenerStruct) CloseConn(conn net.Conn) {
	//activeconn
	conn.Close()

	//plugin
	log.Println("server closed")
}

func (l *ListenerStruct) Close() {
	if l.nl != nil {
		l.nl.Close()
	}
	l.closeDoneChan()
}

func (l *ListenerStruct) Shutdown() {
	atomic.CompareAndSwapInt32(&l.shutdown, 0, 1)
	for {
		if atomic.LoadInt32(&l.handlingNum) == 0 {
			break
		}
	}
	l.closeDoneChan()
	log.Println("server shutdown")
}

//是否处于关闭流程
func (l *ListenerStruct) isShutdown() bool {
	return atomic.LoadInt32(&l.shutdown) == 1
}
