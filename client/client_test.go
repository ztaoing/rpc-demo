/**
* @Author:zhoutao
* @Date:2020/12/30 下午12:49
* @Desc:
 */

package client

import (
	"context"
	"fmt"
	"github.com/ztaoing/rpc-demo/codec"
	"net"
	"strings"
	"testing"
	"time"
)

func _assert(condition bool, msg string, v ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assertion failed:"+msg, v...))
	}
}

// test timeout occured and no time limit
func TestClient_dialTimeout(t *testing.T) {
	//并行
	t.Parallel()
	l, err := net.Listen("tcp", ":0")

	f := func(conn net.Conn, opt *codec.Option) (client *Client, err error) {
		_ = conn.Close()
		time.Sleep(time.Second * 20)
		return nil, nil
	}

	t.Run("timeout", func(t *testing.T) {
		_, err := dialTimeout(f, "tcp", l.Addr().String(), &codec.Option{ConnectionTimeout: time.Second})
		_assert(err != nil, "expect a timeout error")
	})

	t.Run("no_time_limit", func(t *testing.T) {
		_, err = dialTimeout(f, "tcp", l.Addr().String(), &codec.Option{ConnectionTimeout: 0})
		_assert(err == nil, "0 means no time limit")
	})
}

// test handle timeout
type Bar int

func (b Bar) Timeout(argv int, reply *int) error {
	time.Sleep(time.Second * 5)
	return nil
}

func startServer(addr chan string) {
	var b Bar
	_ = codec.Register(&b)
	l, _ := net.Listen("tcp", ":0")
	addr <- l.Addr().String()
	codec.Accept(l)
}

func TestClient_Call(t *testing.T) {
	t.Parallel()
	addrCh := make(chan string)
	go startServer(addrCh)

	addr := <-addrCh
	time.Sleep(time.Second)

	t.Run("client timeout", func(t *testing.T) {
		client, err := Dial("tcp", addr)
		ctx, _ := context.WithTimeout(context.Background(), time.Second)

		var reply int
		err = client.Call(ctx, "Bar.Timeout", 1, &reply)
		_assert(err != nil && strings.Contains(err.Error(), ctx.Err().Error()), "expected a timeout error")
	})

	t.Run("server handle timeout", func(t *testing.T) {
		client, _ := Dial("tcp", addr, &codec.Option{
			HandleTimeout: time.Second,
		})
		var reply int
		err := client.Call(context.Background(), "Bar.Timeout", 1, &reply)
		_assert(err != nil, "expect a timeout error")
	})
}
