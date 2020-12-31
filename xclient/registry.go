/**
* @Author:zhoutao
* @Date:2020/12/31 下午3:12
* @Desc:
 */

package xclient

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type RegistryDiscovery struct {
	*MultiServerDiscover
	registry   string
	timeout    time.Duration
	lastUpdate time.Time
}

const defaultUpdateTimeout = time.Second * 10

func NewRegistryDiscovery(registerAddr string, timeout time.Duration) *RegistryDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	Rd := &RegistryDiscovery{
		MultiServerDiscover: NewMultiServerDiscover(make([]string, 0)),
		registry:            registerAddr,
		timeout:             timeout,
	}
	return Rd
}

func (rd *RegistryDiscovery) Refresh() error {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	if rd.lastUpdate.Add(rd.timeout).After(time.Now()) {
		return nil
	}

	log.Println("rpc registry : refresh servers from registry", rd.registry)
	resp, err := http.Get(rd.registry)
	if err != nil {
		log.Println("rpc registry refresh error : ", err)
		return err
	}
	servers := strings.Split(resp.Header.Get("X-Rpc-Servers"), ",")
	rd.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			rd.servers = append(rd.servers, strings.TrimSpace(server))
		}
	}
	rd.lastUpdate = time.Now()
	return nil
}

func (rd *RegistryDiscovery) Get(mode SelectMode) (string, error) {
	if err := rd.Refresh(); err != nil {
		return "", err
	}
	return rd.MultiServerDiscover.Get(mode)
}

func (rd *RegistryDiscovery) GetAll() ([]string, error) {
	if err := rd.Refresh(); err != nil {
		return nil, err
	}
	return rd.MultiServerDiscover.GetAll()
}
