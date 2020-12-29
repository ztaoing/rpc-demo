/**
* @Author:zhoutao
* @Date:2020/12/27 下午4:37
* @Desc:
 */

package codec

import (
	"encoding/json"
	"fmt"
	"github.com/ztaoing/rpc-demo/codec/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNum = 0x3bef5c

// use json to the Default CODEC of Option, and use CodecType to encoding the left datas
type Option struct {
	MagicNumber int        //means this is a rpc reuqest
	CodecType   codec.Type //the CODEC of a client with the server to encoding datas
}

var DefaultOption = &Option{
	MagicNumber: MagicNum,
	CodecType:   codec.GobCodecType,
}

type Server struct {
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

	s.serveCodec(f(conn))

}

var invalidRequest = struct{}{}

//read request -> handle request -> send Response
func (s *Server) serveCodec(cc codec.Codec) {
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
		go s.handleRequest(cc, req, sendMutex, wg)
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

func (s *Server) readRequest(cc codec.Codec) (*request, error) {
	header, err := s.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	//TODO:
	req := &request{
		header: header,
	}
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server:read argv error:", err)
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

func (s *Server) handleRequest(cc codec.Codec, req *request, sendMutex *sync.Mutex, wg *sync.WaitGroup) {
	//TODO: call registered rpc method to get the reply

	defer wg.Done()
	log.Println(req.header, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("rpc response %d", req.header.Seq))
	s.sendResponse(cc, req.header, req.replyv.Interface(), sendMutex)
}
