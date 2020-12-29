/**
* @Author:zhoutao
* @Date:2020/12/27 下午8:05
* @Desc:
 */

package main

import (
	"encoding/json"
	"fmt"
	"github.com/ztaoing/rpc-demo/codec"
	codec2 "github.com/ztaoing/rpc-demo/codec/codec"
	"log"
	"net"
	"time"
)

func startServer(addr chan string) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	codec.Accept(l)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)

	// rpc client
	conn, _ := net.Dial("tcp", <-addr)
	defer func() {
		_ = conn.Close()
	}()

	time.Sleep(time.Second * 5)

	//send options to
	_ = json.NewEncoder(conn).Encode(codec.DefaultOption)

	cc := codec2.NewGobCodec(conn)

	//send request and receive response
	for i := 0; i < 5; i++ {
		h := &codec2.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		_ = cc.Write(h, fmt.Sprintf("rpc req %d", h.Seq))

		//receive response
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Printf("reply", reply)
	}
}
