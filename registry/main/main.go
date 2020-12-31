/**
* @Author:zhoutao
* @Date:2020/12/31 下午3:38
* @Desc:
 */

package main

import (
	"github.com/ztaoing/rpc-demo/codec"
	"github.com/ztaoing/rpc-demo/registry"
	"net"
	"net/http"
	"sync"
)

type Foo int

type Args struct {
	Num1, Num2 int
}

func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":9999")
	registry.HandleHTTP()

	wg.Done()
	_ = http.Serve(l, nil)
}

func startServer(registryAddr string, wg *sync.WaitGroup) {
	var foo Foo
	l, _ := net.Listen("tcp", ":0")
	server := codec.NewServer()
	_ = server.Register(&foo)
	registry.HeartBeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	wg.Done()

	server.Accept(l)
}
