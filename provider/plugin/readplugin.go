package plugin

import (
	"fmt"
	"github.com/LJJsde/lrpc/protocol"
)

type BeforeReadPlugin struct{}

func (p BeforeReadPlugin) BeforeRead() error {
	fmt.Println("==== before read plugin ====")
	return nil
}

type AfterReadPlugin struct{}

func (p AfterReadPlugin) AfterRead(msg *protocol.MessageStruct, err error) error {
	fmt.Println("==== after read plugin ====", msg, err)
	return nil
}
