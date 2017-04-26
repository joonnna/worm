package communication

import (
	"fmt"
	"github.com/joonnna/worm/util"
	"io"
	"io/ioutil"
	"log"
	"net"
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

	UdpConn net.PacketConn

	client *http.Client

	allHostsMutex    sync.RWMutex
	activeHostsMutex sync.RWMutex
}

func Init_comm(HostName, hostPort, wormgatePort string) *Comm {
	c := &Comm{
		HostName:     HostName,
		WormgatePort: wormgatePort,
		HostPort:     hostPort,
		client:       &http.Client{},
	}

	return c
}

func (c *Comm) CommStatus(ch chan<- bool) {
	startup := true

	for {
		allHosts := util.FetchReachableHosts(c.WormgatePort, c.HostName)
		currHosts := c.GetAllHosts()

		diff := util.SliceDiff(currHosts, allHosts)
		if len(diff) > 0 {
			c.setAllHosts(allHosts)
		}

		active := c.PingHosts()

		activeDiff := util.SliceDiff(c.GetActiveHosts(), active)

		if len(activeDiff) > 0 {
			c.setActiveHosts(active)
		}

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

func (c *Comm) PingHosts() []string {
	hosts := c.GetAllHosts()

	var activeHosts []string

	ch := make(chan string, len(activeHosts))

	for _, addr := range hosts {
		go c.doPing(addr, ch)
	}

	for i := 0; i < len(hosts); i++ {
		host := <-ch
		if host != "" {
			activeHosts = append(activeHosts, host)
		}
	}
	return activeHosts
}

func (c *Comm) InitUdp() {

	conn, err := net.ListenPacket("udp", c.HostName+":12332")
	if err != nil {
		log.Panic("Cant start udp")
	}

	c.UdpConn = conn

	data := make([]byte, 1024)

	for {
		_, addr, err := conn.ReadFrom(data)
		if err != nil {
			fmt.Println(err)
		} else {
			_, err = conn.WriteTo([]byte("alive"), addr)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (c Comm) udpPing(addr string, ch chan<- string) {

	udpAddr, err := net.ResolveUDPAddr("udp", addr+":12332")
	if err != nil {
		ch <- ""
		fmt.Println(err)
		return
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		ch <- ""
		return
	}
	defer conn.Close()
	_, err = conn.Write([]byte("JAVELL DA"))
	if err != nil {
		ch <- ""
		return
	}

	data := make([]byte, 128)

	t := time.Now()
	conn.SetReadDeadline(t.Add(time.Millisecond * 1))

	bytes, err := conn.Read(data)
	if err == nil {
		if bytes > 0 {
			ch <- addr
		} else {
			ch <- ""
		}
	} else {
		ch <- ""
	}

}

func (c Comm) doPing(addr string, ch chan<- string) {
	url := fmt.Sprintf("http://%s%s/alive", addr, c.HostPort)
	resp, err := c.client.Get(url)
	if err != nil {
		ch <- ""
		return
	}

	if resp.StatusCode == 200 || resp.StatusCode == 409 {
		ch <- addr
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	ch <- ""
}

func (c *Comm) setAllHosts(hosts []string) {
	c.allHostsMutex.Lock()
	c.allHosts = hosts
	c.allHostsMutex.Unlock()
}

func (c *Comm) GetAllHosts() []string {
	c.allHostsMutex.RLock()
	ret := c.allHosts
	c.allHostsMutex.RUnlock()

	return ret
}

func (c *Comm) setActiveHosts(hosts []string) {
	c.activeHostsMutex.Lock()
	c.activeHosts = hosts
	c.activeHostsMutex.Unlock()
}

func (c *Comm) GetActiveHosts() []string {
	c.activeHostsMutex.RLock()
	ret := c.activeHosts
	c.activeHostsMutex.RUnlock()

	return ret
}