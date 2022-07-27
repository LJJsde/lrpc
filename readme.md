# RPC框架

## 使用
### 开启服务中心
 ```go
  cd center/cmd
  go build -o discovery main.go
  go run main.go -c configs.yaml
 ```
![122](https://user-images.githubusercontent.com/32420724/181150772-fc4c2215-0584-4c01-b593-ae0da7e0264a.PNG)

### 启动服务端
```go
 cd demo
 go run server.go -c config.yaml
 ```
![123](https://user-images.githubusercontent.com/32420724/181150781-7d0dd54b-0cfa-40e7-91a2-35121b49debb.PNG)

### 客户端示例
```go
func main() {
	nodes := []string{"localhost:8881"}
	conf := &naming.Config{Nodes: nodes, Env: "dev"}
	discovery := naming.New(conf)

	gob.Register(User{})
	cli := consumer.CreateClientProxy("UserService", consumer.DefaultOption, discovery)

	var GetUserById func(id int) (User, error)

	//wrap call
	ret, err := cli.Call(context.Background(), "User.GetUserById", &GetUserById, 1)
	if err != nil {
		log.Println("call error:", err)
	} else {
		val := ret.([]reflect.Value)
		user := val[0].Interface().(User)
		log.Println("rpc return result:", user)
	}

	//makefunc and call
}
```


