/**
* @Author:zhoutao
* @Date:2020/12/30 上午10:42
* @Desc:
 */

package main

import (
	"github.com/ztaoing/rpc-demo/client"
	"github.com/ztaoing/rpc-demo/codec"
	"log"
	"net"
	"sync"
	"time"
)

type Foo int

type Args struct {
	Num1, Num2 int
}

// a exported method
func (f Foo) Sum(args Args, replay *int) error {
	*replay = args.Num1 + args.Num2
	return nil
}

// register Foo to server
func startServer(addr chan string) {
	var foo Foo
	if err := codec.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on:", l.Addr())
	addr <- l.Addr().String()
	codec.Accept(l)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)
	c, _ := client.Dial("tcp", <-addr)
	defer func() {
		c.Close()
	}()

	time.Sleep(time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i * i}

			var reply int
			if err := c.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error :", err)
			}
			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}
