/**
* @Author:zhoutao
* @Date:2020/12/31 上午11:20
* @Desc:
 */

package main

import (
	"context"
	"github.com/ztaoing/rpc-demo/codec"
	"github.com/ztaoing/rpc-demo/xclient"
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

//timeout
func (f Foo) Sleep(args Args, reply *int) error {
	time.Sleep(time.Second * time.Duration(args.Num1))
	*reply = args.Num1 + args.Num2
	return nil
}

func startServer(addrChan chan string) {
	var foo Foo
	l, _ := net.Listen("tcp", ":0")
	server := codec.NewServer()
	_ = server.Register(&foo)
	addrChan <- l.Addr().String()
	server.Accept(l)
}

func foo(c *xclient.XClient, ctx context.Context, typ, serviceMethod string, args *Args) {
	var reply int
	var err error

	switch typ {
	case "call":
		err = c.Call(ctx, serviceMethod, args, reply)
	case "broadcast":
		err = c.Broadcast(ctx, serviceMethod, args, reply)
	}
	if err != nil {
		log.Printf("%s %s error : %v", typ, serviceMethod, err)
	} else {
		log.Printf("%s %s success : %d + %d = %d ", typ, serviceMethod, args.Num1, args.Num2, reply)
	}
}

func call(addr1, addr2 string) {
	d := xclient.NewMultiServerDiscover([]string{"tcp@" + addr1, "tcp@" + addr2})
	c := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() {
		_ = c.Close()
	}()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(c, context.Background(), "call", "Foo.Sum", &Args{Num1: i, Num2: i * i})
		}(i)

	}
	wg.Wait()
}

func broadcast(addr1, addr2 string) {
	d := xclient.NewMultiServerDiscover([]string{"tcp@" + addr1, "tcp@" + addr2})
	c := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() {
		_ = c.Close()
	}()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// no time limit
			//foo(c, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			// expect timeout
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			foo(c, ctx, "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
		}(i)

	}
	wg.Wait()
}

func main() {
	log.SetFlags(0)
	ch1 := make(chan string)
	ch2 := make(chan string)

	go startServer(ch1)
	go startServer(ch2)

	addr1 := <-ch1
	addr2 := <-ch2

	time.Sleep(time.Second)

	call(addr1, addr2)
	broadcast(addr1, addr2)

}
