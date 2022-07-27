package consumer

import (
	"context"
	"errors"
	//"fmt"
	"github.com/LJJsde/lrpc/config"
	"github.com/LJJsde/lrpc/protocol"
	"log"
	"net"
	"reflect"
	"time"
	"github.com/LJJsde/lrpc/codec"
)

var ParamErr = errors.New("params not adapted")

type Client interface {
	Connect(string) error
	//直接执行
	Invoke(context.Context, *Service, interface{}, ...interface{}) (interface{}, error)
	//构造执行方法
	MakeFunc(*Service, interface{})
	Close()
	GetAddr() string
}

type Option struct {
	Retries           int
	FailMode          FailMode
	ConnectionTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	SerializeType     protocol.SerializeType
	CompressType      protocol.CompressType
	NetProtocol       string
	LoadBalanceMode   LoadBalanceMode
}

var DefaultOption = Option{
	Retries:           3,
	FailMode:          Failover,
	ConnectionTimeout: 5 * time.Second,
	ReadTimeout:       1 * time.Second,
	WriteTimeout:      1 * time.Second,
	SerializeType:     protocol.Gob,
	CompressType:      protocol.None,
	NetProtocol:       "tcp",
	LoadBalanceMode:   RoundRobinBalance,
}

type ClientStruct struct {
	conn   net.Conn
	option Option
	addr   string
}

func CreateClient(option Option) Client {
	return &ClientStruct{option: option}
}

func (cli *ClientStruct) Connect(addr string) error {
	conn, err := net.DialTimeout(cli.option.NetProtocol, addr, cli.option.ConnectionTimeout)
	if err != nil {
		return err
	}
	cli.conn = conn
	cli.addr = addr
	return nil
}

func (cli *ClientStruct) Invoke(ctx context.Context, service *Service, stub interface{}, params ...interface{}) (interface{}, error) {
	//make func : this step can be prepared before invoke and store into cache
	cli.MakeFunc(service, stub)
	//reflect call
	return cli.wrapCall(ctx, stub, params...)
}

func (cli *ClientStruct) Close() {
	if cli.conn != nil {
		cli.conn.Close()
	}
}

func (cli *ClientStruct) GetAddr() string {
	//cli.conn.RemoteAddr().String()
	return cli.addr
}

//make call func
func (cli *ClientStruct) MakeFunc(service *Service, methodPtr interface{}) {
	container := reflect.ValueOf(methodPtr).Elem() //反射获取函数元素
	coder := codec.Codecs[cli.option.SerializeType]

	handler := func(req []reflect.Value) []reflect.Value {
		//出参个数
		numOut := container.Type().NumOut()

		//error
		errorHandler := func(err error) []reflect.Value {
			outArgs := make([]reflect.Value, numOut)
			for i := 0; i < len(outArgs)-1; i++ {
				outArgs[i] = reflect.Zero(container.Type().Out(i))
			}
			outArgs[len(outArgs)-1] = reflect.ValueOf(&err).Elem()
			return outArgs
		}

		//in args
		inArgs := make([]interface{}, 0, len(req))
		for _, arg := range req {
			inArgs = append(inArgs, arg.Interface())
		}

		payload, err := coder.Encode(inArgs) //[]byte
		if err != nil {
			log.Printf("encode err:%v\n", err)
			return errorHandler(err)
		}

		//send by network
		startTime := time.Now()
		if cli.option.WriteTimeout != 0 {
			cli.conn.SetWriteDeadline(startTime.Add(cli.option.WriteTimeout))
		}
		msg := protocol.NewMessage()
		msg.SetVersion(config.Protocol_MsgVersion)
		msg.SetMsgType(protocol.Request)
		msg.SetCompressType(cli.option.CompressType)
		msg.SetSerializeType(cli.option.SerializeType)
		msg.ServiceClass = service.Class
		msg.ServiceMethod = service.Method
		msg.Payload = payload
		err = msg.Send(cli.conn)
		if err != nil {
			log.Printf("send err:%v\n", err)
			return errorHandler(err)
		}
		log.Println("send success!")

		//read from network
		if cli.option.ReadTimeout != 0 {
			cli.conn.SetReadDeadline(startTime.Add(cli.option.ReadTimeout))
		}
		respMsg, err := protocol.Read(cli.conn)
		if err != nil {
			return errorHandler(err)
		}
		log.Println("response success!")

		//decode response
		respDecode := make([]interface{}, 0)
		err = coder.Decode(respMsg.Payload, &respDecode)
		if err != nil {
			log.Printf("decode err:%v\n", err)
			return errorHandler(err)
		}
		log.Println("decode success!")

		//output result
		if len(respDecode) == 0 {
			respDecode = make([]interface{}, numOut)
		}
		outArgs := make([]reflect.Value, numOut)
		for i := 0; i < numOut; i++ {
			if i != numOut { //处理非error
				if respDecode[i] == nil {
					outArgs[i] = reflect.Zero(container.Type().Out(i))
				} else {
					outArgs[i] = reflect.ValueOf(respDecode[i])
				}
			} else { //处理error
				outArgs[i] = reflect.Zero(container.Type().Out(i))
			}
		}
		return outArgs
	}

	container.Set(reflect.MakeFunc(container.Type(), handler)) //构造函数
}

func (cli *ClientStruct) wrapCall(ctx context.Context, stub interface{}, params ...interface{}) (interface{}, error) {
	f := reflect.ValueOf(stub).Elem()
	if len(params) != f.Type().NumIn() {
		return nil, ParamErr
	}

	in := make([]reflect.Value, len(params))
	for idx, param := range params {
		in[idx] = reflect.ValueOf(param)
	}
	result := f.Call(in)
	return result, nil
}
