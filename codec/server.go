/**
* @Author:zhoutao
* @Date:2020/12/27 下午4:37
* @Desc:
 */

package codec

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ztaoing/rpc-demo/codec/codec"
	"github.com/ztaoing/rpc-demo/service"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

const MagicNum = 0x3bef5c

// use json to the Default CODEC of Option, and use CodecType to encoding the left datas
type Option struct {
	MagicNumber       int           //means this is a rpc reuqest
	CodecType         codec.Type    //the CODEC of a client with the server to encoding datas
	ConnectionTimeout time.Duration // 0 means  no time limit
	HandleTimeout     time.Duration
}

var DefaultOption = &Option{
	MagicNumber:       MagicNum,
	CodecType:         codec.GobCodecType,
	ConnectionTimeout: time.Second * 10,
}

type Server struct {
	ServiceMap sync.Map
}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func Accept(l net.Listener) {
	DefaultServer.Accept(l)
}

func (s *Server) Accept(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("rpc server:accect() error:", err)
			return
		}
		go s.ServeConn(conn)
	}
}

func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() {
		_ = conn.Close()
	}()

	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: decoding Option by json error:", err)
		return
	}
	// rpc request
	if opt.MagicNumber != MagicNum {
		log.Printf("rpc server: needs rpc request %s:", opt.CodecType)
		return
	}

	// get func by Codec.type from map
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}

	s.serveCodec(f(conn), &opt)

}

var invalidRequest = struct{}{}

//read request -> handle request -> send Response
func (s *Server) serveCodec(cc codec.Codec, opt *Option) {
	sendMutex := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		req, err := s.readRequest(cc)
		if err != nil {
			if req == nil {
				break // get nothing and  can not recover,so close the connection
			}
			req.header.Error = err.Error()
			s.sendResponse(cc, req.header, invalidRequest, sendMutex)
		}
		//success
		wg.Add(1)
		go s.handleRequest(cc, req, sendMutex, wg, opt.HandleTimeout)
	}
	wg.Wait()
	_ = cc.Close()
}

func (s *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var header codec.Header
	if err := cc.ReadHeader(&header); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &header, nil
}

// read header , then decode body with CODEC
func (s *Server) readRequest(cc codec.Codec) (*request, error) {
	header, err := s.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{
		header: header,
	}

	req.svc, req.mType, err = s.findService(header.ServiceMethod)
	if err != nil {
		return req, err
	}

	req.argv = req.mType.NewArgv()
	req.replyv = req.mType.NewReplyv()
	//argv may be pointer or  value type
	argvi := req.argv.Interface()
	//argvi must be a pointer
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	// use CODEC to decode the body
	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server:read body err:", err)
		return req, err
	}
	return req, nil
}

func (s *Server) sendResponse(cc codec.Codec, header *codec.Header, body interface{}, sendMutex *sync.Mutex) {
	sendMutex.Lock()
	defer sendMutex.Unlock()

	if err := cc.Write(header, body); err != nil {
		log.Println("rpc server:write response error:", err)
	}
}

func (s *Server) handleRequest(cc codec.Codec, req *request, sendMutex *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()
	//sendResponse has two parts: called and send , the two parts to make sure the sendResponse called only one time
	//have called the service.method
	called := make(chan struct{})
	//have sendResponse to client
	send := make(chan struct{})

	go func() {
		err := req.svc.Call(req.mType, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.header.Error = err.Error()
			s.sendResponse(cc, req.header, req.replyv.Interface(), sendMutex)
			send <- struct{}{}
			return
		}
		s.sendResponse(cc, req.header, req.replyv.Interface(), sendMutex)
		send <- struct{}{}
	}()
	// waiting with block
	if timeout == 0 {
		<-called
		<-send
		return
	}
	select {
	case <-time.After(timeout):
		req.header.Error = fmt.Sprintf("rpc server:  handle request timeout:expected within %s", timeout.String())
		s.sendResponse(cc, req.header, req.replyv.Interface(), sendMutex)
	case <-called:
		<-send
	}

}

func (s *Server) Register(rcvr interface{}) error {
	Service := service.NewService(rcvr)
	if _, dup := s.ServiceMap.LoadOrStore(Service.Name, Service); dup {
		return errors.New("rpc Service already defiend: " + Service.Name)
	}
	return nil
}

func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}

func (s *Server) findService(ServiceMethod string) (svc *service.Service, mType *service.MethodType, err error) {
	dot := strings.LastIndex(ServiceMethod, ".")
	// if .  dose not in the string of ServiceMethod ,return -1
	if dot < 0 {
		err = errors.New("rpc Service : Service/method is ill-formed:" + ServiceMethod)
		return
	}
	ServiceName, MethodName := ServiceMethod[:dot], ServiceMethod[dot+1:]
	svcLoad, ok := s.ServiceMap.Load(ServiceName)
	if !ok {
		err = errors.New("rpc server : can not find Service :" + ServiceName)
		return
	}
	svc = svcLoad.(*service.Service)
	mType = svc.Method[MethodName]
	if mType == nil {
		err = errors.New("rpc server :can not find method :" + MethodName)
		return
	}
	return
}
