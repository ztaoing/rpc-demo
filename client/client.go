/**
* @Author:zhoutao
* @Date:2020/12/28 下午3:15
* @Desc:
 */

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	codec2 "github.com/ztaoing/rpc-demo/codec"
	"github.com/ztaoing/rpc-demo/codec/codec"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Call struct {
	Seq           uint64
	ServiceMethod string
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call
}

func (c *Call) done() {
	c.Done <- c
}

type Client struct {
	cc        codec.Codec
	opt       *codec2.Option
	sendMutex sync.Mutex
	header    codec.Header
	mu        sync.Mutex
	seq       uint64
	pending   map[uint64]*Call
	closing   bool //user close
	shutdown  bool //server close
}

func NewClient(conn net.Conn, opt *codec2.Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client:codec error:", err)
		return nil, err
	}
	//send option to server
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client:options error:", err)
		_ = conn.Close()
		return nil, err
	}
	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc codec.Codec, opt *codec2.Option) *Client {
	client := &Client{
		seq:     1,
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

var _ io.Closer = (*Client)(nil)
var ErrShutdown = errors.New("connection has been shuted down")

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing {
		return ErrShutdown
	}

	c.closing = true
	return c.cc.Close()
}

func (c *Client) IsAvailable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.shutdown && !c.closing
}

func (c *Client) registerCall(call *Call) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closing || c.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = c.seq
	c.pending[call.Seq] = call
	c.seq++ //prepare for next reggister
	return call.Seq, nil
}

func (c *Client) removeCall(seq uint64) *Call {
	c.mu.Lock()
	defer c.mu.Unlock()
	call := c.pending[seq]
	delete(c.pending, seq)
	return call
}

// when client or server ocurs error,then termindate server and send error to every call
func (c *Client) terminateCalls(err error) {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()

	c.shutdown = true
	for _, call := range c.pending {
		call.Error = err
		call.done()
	}
}

// receive response from server ,3 satiation:
// 1 : request is not complete
// 2 : request is complete ,but server answer err
// 3 : request is complete ,and server answer right
func (c *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header
		if err = c.cc.ReadHeader(&h); err != nil {
			break
		}
		call := c.removeCall(h.Seq)
		switch {
		case call == nil:
			err = c.cc.ReadBody(nil)
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = c.cc.ReadBody(nil)
			call.done()

		default:
			err = c.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body" + err.Error())
			}
			call.done()
		}

	}
	// error occurs, so terminate server and send error mesage to every call
	c.terminateCalls(err)
}

func (c *Client) send(call *Call) {
	c.sendMutex.Lock()
	defer c.sendMutex.Unlock()

	//register call
	seq, err := c.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	//header
	c.header.ServiceMethod = call.ServiceMethod
	c.header.Seq = seq
	c.header.Error = ""

	//encode with CODEC and send the request
	if err = c.cc.Write(&c.header, call.Args); err != nil {
		//call may be nil,
		call := c.removeCall(c.seq)
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

func (c *Client) Go(ServiceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)

	} else if cap(done) == 0 {
		log.Panic("rpc client:done channel is full")
	}
	call := &Call{
		ServiceMethod: ServiceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	c.send(call)
	return call
}

func (c *Client) Call(ctx context.Context, ServiceMethod string, args, reply interface{}) error {

	//waiting for the response
	call := c.Go(ServiceMethod, args, reply, make(chan *Call, 1))
	select {
	case <-ctx.Done():
		c.removeCall(call.Seq)
		return errors.New("rpc client failed:" + ctx.Err().Error())
	case done := <-call.Done:
		return done.Error
	}

}

func parseOptions(opts ...*codec2.Option) (*codec2.Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return codec2.DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of the options can not be more than 1")
	}

	opt := opts[0]
	opt.MagicNumber = codec2.DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = codec2.DefaultOption.CodecType
	}
	return opt, nil
}

func Dial(network, address string, opts ...*codec2.Option) (client *Client, err error) {
	// dial to server with time limit
	return dialTimeout(NewClient, network, address, opts...)
}

type clientResult struct {
	client *Client
	err    error
}

type newClientFunc func(conn net.Conn, opt *codec2.Option) (client *Client, err error)

func dialTimeout(f newClientFunc, network, address string, opts ...*codec2.Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTimeout(network, address, opt.ConnectionTimeout)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()

	ch := make(chan clientResult)
	go func() {
		c, err := f(conn, opt)
		ch <- clientResult{
			client: c,
			err:    err,
		}
	}()

	// no time limit
	if opt.ConnectionTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}

	select {
	case <-time.After(opt.ConnectionTimeout):
		return nil, fmt.Errorf("rpc client :connect to server timeout : expected within %s", opt.ConnectionTimeout.String())
	case result := <-ch:
		return result.client, result.err
	}

}
