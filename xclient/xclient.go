/**
* @Author:zhoutao
* @Date:2020/12/30 下午3:21
* @Desc:
 */

package xclient

import (
	"context"
	"github.com/ztaoing/rpc-demo/client"
	"github.com/ztaoing/rpc-demo/codec"
	"io"
	"reflect"
	"sync"
)

type XClient struct {
	discover Discover
	mode     SelectMode
	opt      *codec.Option
	mu       sync.Mutex
	clients  map[string]*client.Client
}

func NewXClient(d Discover, mode SelectMode, opt *codec.Option) *XClient {
	return &XClient{discover: d, mode: mode, opt: opt, clients: make(map[string]*client.Client)}
}

func (x *XClient) Close() error {
	x.mu.Lock()
	defer x.mu.Unlock()
	//close and delete
	for key, client := range x.clients {
		_ = client.Close()
		delete(x.clients, key)
	}
	return nil
}

var _ io.Closer = (*XClient)(nil)

func (x *XClient) dialDiscover(rpcAddr string) (*client.Client, error) {
	x.mu.Lock()
	defer x.mu.Unlock()

	c, ok := x.clients[rpcAddr]
	//rpcAddr is closed
	if ok && !c.IsAvailable() {
		_ = c.Close()
		delete(x.clients, rpcAddr)
		// ?
		c = nil
	}
	if c == nil {
		var err error
		c, err = client.XDial(rpcAddr, x.opt)
		if err != nil {
			return nil, err
		}
		x.clients[rpcAddr] = c
	}
	return c, nil
}

func (x *XClient) callDiscover(rpcAddr string, ctx context.Context, serviceMethod string, args, reply interface{}) error {
	c, err := x.dialDiscover(rpcAddr)
	if err != nil {
		return err
	}
	//call remote Service.Method
	return c.Call(ctx, serviceMethod, args, reply)
}

func (x *XClient) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	// get rpcAddr from registry
	rpcAddr, err := x.discover.Get(x.mode)
	if err != nil {
		return err
	}
	return x.callDiscover(rpcAddr, ctx, serviceMethod, args, reply)
}

// broadcast the request to  all server
func (x *XClient) Broadcast(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	servers, err := x.discover.GetAll()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var e error
	replyDone := reply == nil

	ctx, cancel := context.WithCancel(ctx)
	for _, rpcAddr := range servers {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var cloneReply interface{}

			if reply != nil {
				cloneReply = reflect.New(reflect.ValueOf(&reply).Elem().Type()).Interface()
			}
			err = x.callDiscover(rpcAddr, ctx, serviceMethod, args, reply)

			mu.Lock()
			if err != nil && e == nil {
				e = err
				// after few successful response occurs one failed response,the e will be set ,
				// the replyDone  need to be set false

				replyDone = false
				// if  any call failed, cancel all unfinished calls
				cancel()

			}
			// make sure the replayDone only can be set once by the successful response
			if err == nil && !replyDone {
				reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(cloneReply).Elem())
				replyDone = true
			}
			mu.Unlock()

		}(rpcAddr)
	}
	wg.Wait()
	return e
}
