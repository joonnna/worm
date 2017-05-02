package communication

import (
	"github.com/joonnna/worm/util"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type Comm struct {
	HostName     string
	HostPort     string
	WormgatePort string

	allHosts    []string
	activeHosts []string

	client *http.Client

	allHostsMutex    sync.RWMutex
	activeHostsMutex sync.RWMutex

	Transmission
}

type Transmission interface {
	PingHosts() []string
}

func InitComm(HostName, hostPort, wormgatePort string, t Transmission) *Comm {
	c := &Comm{
		HostName:     HostName,
		WormgatePort: wormgatePort,
		HostPort:     hostPort,
		client:       &http.Client{Timeout: time.Second * 2},
		Transmission: t,
	}

	return c
}

func (c *Comm) CommStatus(ch chan<- bool) {
	startup := true

	for {
		allHosts := util.FetchReachableHosts(c.WormgatePort, c.HostName)

		c.setAllHosts(allHosts)

		active := c.PingHosts()

		c.setActiveHosts(active)

		if startup {
			ch <- true
			startup = false
		}
	}
}

func (c Comm) ContactHostHttp(req *http.Request) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	return err
}

func (c *Comm) setAllHosts(hosts []string) {
	c.allHostsMutex.Lock()

	tmp := make([]string, len(hosts))
	copy(tmp, hosts)

	c.allHosts = tmp

	//c.allHosts = hosts
	c.allHostsMutex.Unlock()
}

func (c *Comm) GetAllHosts() []string {
	c.allHostsMutex.RLock()
	ret := make([]string, len(c.allHosts))
	copy(ret, c.allHosts)
	c.allHostsMutex.RUnlock()

	return ret
}

func (c *Comm) setActiveHosts(hosts []string) {
	c.activeHostsMutex.Lock()

	tmp := make([]string, len(hosts))

	copy(tmp, hosts)

	c.activeHosts = tmp
	c.activeHostsMutex.Unlock()
}

func (c *Comm) GetActiveHosts() []string {
	c.activeHostsMutex.RLock()

	ret := make([]string, len(c.activeHosts))
	copy(ret, c.activeHosts)

	//ret := c.activeHosts
	c.activeHostsMutex.RUnlock()

	return ret
}
