/**
* @Author:zhoutao
* @Date:2020/12/30 下午3:14
* @Desc:
 */

package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type SelectMode int

const (
	RandomSelect SelectMode = iota
	RoundRobinSelect
)

type Discover interface {
	Refresh() error
	Update(servers []string) error
	Get(mode SelectMode) (string, error)
	GetAll() ([]string, error)
}

//without registry center
type MultiServerDiscover struct {
	r       *rand.Rand
	mu      sync.RWMutex
	servers []string
	index   int
}

func (m *MultiServerDiscover) Refresh() error {
	return nil
}

func (m *MultiServerDiscover) Update(servers []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.servers = servers
	return nil
}

//get alive server with selectMode
func (m *MultiServerDiscover) Get(mode SelectMode) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	n := len(m.servers)
	if n == 0 {
		return "", errors.New("rpc discover error: no available servser")
	}

	switch mode {
	case RandomSelect:
		return m.servers[m.r.Intn(n)], nil
	case RoundRobinSelect:
		s := m.servers[m.index%n]
		m.index = (m.index + 1) % n
		return s, nil
	default:
		return "", errors.New("rpc discover error : not supported select mode")
	}
}

func (m *MultiServerDiscover) GetAll() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// return a copy of m.servers
	servers := make([]string, len(m.servers), len(m.servers))
	copy(servers, m.servers)
	return servers, nil
}

func NewMultiServerDiscover(servers []string) *MultiServerDiscover {
	d := &MultiServerDiscover{
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
		servers: servers,
	}
	d.index = d.r.Intn(math.MaxInt32 - 1)
	return d
}

var _ Discover = (*MultiServerDiscover)(nil)
