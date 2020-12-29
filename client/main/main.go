/**
* @Author:zhoutao
* @Date:2020/12/28 下午8:31
* @Desc:
 */

package main

import (
	"fmt"
	"github.com/ztaoing/rpc-demo/client"
	codec2 "github.com/ztaoing/rpc-demo/codec"
	"log"
	"net"
	"sync"
	"time"
)

func startServer(addr chan string) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on:", l.Addr())
	addr <- l.Addr().String()
	codec2.Accept(l)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)

	go startServer(addr)

	c, err := client.Dial("tcp", <-addr)
	defer func() {
		_ = c.Close()
	}()

	time.Sleep(time.Second * 2)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("rpc req %d", i)
			var reply string

			if err = c.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error", err)
			}
			log.Println("replay:", reply)
		}(i)
	}
	wg.Wait()
}
