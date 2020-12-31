/**
* @Author:zhoutao
* @Date:2020/12/31 下午2:01
* @Desc:
 */

package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type RpcRegistry struct {
	timeout time.Duration // all the server address will be timeout at the same time is bad
	mu      sync.Mutex    // read write lock , registry will be changing all the time ,get alive server is important
	servers map[string]*ServerItem
}

type ServerItem struct {
	Addr  string
	start time.Time
}

const (
	defaultPath    = "/rpc_demo/registry"
	defaultTimeout = time.Minute * 5
)

func NewRegistry(timeout time.Duration) *RpcRegistry {
	return &RpcRegistry{
		timeout: timeout,
		servers: make(map[string]*ServerItem),
	}
}

var DefaultRegistry = NewRegistry(defaultTimeout)

func (r *RpcRegistry) PutServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{
			Addr:  addr,
			start: time.Now(),
		}
	} else {
		// update time
		s.start = time.Now()
	}
}

// get alive servers
func (r *RpcRegistry) getServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var alive []string

	for addr, s := range r.servers {
		if r.timeout == 0 || s.start.Add(r.timeout).After(time.Now()) {
			alive = append(alive, addr)
		} else {
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

func (r *RpcRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		w.Header().Set("X-Rpc-Demo-Server", strings.Join(r.getServers(), ","))
	case "POST":
		addr := req.Header.Get("X-Rpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.PutServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// RpcRegistry implemented ServeHTTP()
func (r *RpcRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path : ", registryPath)
}

func HandleHTTP() {
	DefaultRegistry.HandleHTTP(defaultPath)
}

// send a message to server every minute
func HeartBeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		// make sure  it has  enough time to send heart beat  before removed from registry
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartBeat(registry, addr)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartBeat(registry, addr)
		}
	}()

}

func sendHeartBeat(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)

	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Rpc-Server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server :heartbeat err : ", err)
		return err
	}
	return nil
}
