/**
* @Author:zhoutao
* @Date:2020/12/30 下午1:42
* @Desc:
 */

package codec

import (
	"io"
	"log"
	"net/http"
)

const (
	connected        = "200 Connected to RPC"
	defaultRPCPath   = "/_rpcdemo_"
	defaultDebugPath = "/debug/rpc"
)

//ServeHTTP implements an http.handler to answer HTTP requests
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		w.Header().Set("Content-Type", "text/plain;charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, "405 must be CONNECT\n")
		return
	}

	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Print("rpc jijacking", req.RemoteAddr, ":", err.Error())
		return
	}
	_, _ = io.WriteString(conn, "HTTP/1.0 "+connected+"\n\n")
	s.ServeConn(conn)
}

// todo:?
func (s *Server) HandleHTTP() {
	http.Handle(defaultRPCPath, s)
}

func HandleHTTP() {
	DefaultServer.HandleHTTP()
}
