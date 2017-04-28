package communication

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joonnna/worm/util"
	"io"
	"io/ioutil"

	"net"
	"net/http"
	"sync"
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
	GetTargetSegments() int
	GetKillRate() float32
	Ping(addr string, ch chan<- string, data []byte)
}

type Message struct {
	Addr      string
	TargetSeg int
	KillRate  float32
}

func InitComm(HostName, hostPort, wormgatePort string, t Transmission) *Comm {
	c := &Comm{
		HostName:     HostName,
		WormgatePort: wormgatePort,
		HostPort:     hostPort,
		client:       &http.Client{},
		Transmission: t,
	}

	return c
}

func (c *Comm) CommStatus(ch chan<- bool) {
	startup := true

	for {
		allHosts := util.FetchReachableHosts(c.WormgatePort, c.HostName)

		c.setAllHosts(allHosts)

		active := c.pingHosts()

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

func (c *Comm) pingHosts() []string {
	hosts := c.GetAllHosts()

	var activeHosts []string

	ch := make(chan string, len(hosts))

	msg := &Message{
		Addr:      c.HostName,
		TargetSeg: c.GetTargetSegments(),
		KillRate:  c.GetKillRate(),
	}

	marsh, err := json.Marshal(msg)
	if err != nil {
		fmt.Println(err)
	}

	buf := bytes.NewBuffer(marsh)

	err = json.NewEncoder(buf).Encode(msg)
	if err != nil {
		fmt.Println(err)
	}

	data := buf.Bytes()

	for _, addr := range hosts {
		go c.Ping(addr, ch, data)
	}

	for i := 0; i < len(hosts); i++ {
		host := <-ch
		if host != "" {
			activeHosts = append(activeHosts, host)
		}
	}
	return activeHosts
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
