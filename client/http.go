/**
* @Author:zhoutao
* @Date:2020/12/30 下午2:11
* @Desc:
 */

package client

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/ztaoing/rpc-demo/codec"
	"io"
	"net"
	"net/http"
	"strings"
)

const defaultRPCPath = "/_rpcdemo_"

func NewHTTPClient(conn net.Conn, opt *codec.Option) (*Client, error) {
	_, _ = io.WriteString(conn, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", defaultRPCPath))

	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == "200" {
		// after conncted to rpc server with HTTP CONNECT,then use HTTP to connect server
		return NewClient(conn, opt)
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	return nil, err
}

// connect to an HTTP RPC server
func DialHTTP(network, address string, opts ...*codec.Option) (*Client, error) {
	return dialTimeout(NewHTTPClient, network, address, opts...)
}

func XDial(rpcAddr string, opts ...*codec.Option) (*Client, error) {
	//protocol@addr
	parts := strings.Split(rpcAddr, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("rpc client error : wrong format '%s',expect protocol@addr", rpcAddr)
	}
	protocol, addr := parts[0], parts[1]
	switch protocol {
	case "http":
		return DialHTTP("tcp", addr, opts...)
	default:
		// tcp ,unix or other protocol
		return Dial(protocol, addr, opts...)
	}
}
