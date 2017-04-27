package communication

import (
	"bytes"
	"encoding/json"
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

	getTarget func() int
	setTarget func(int)
	getLeader func() string
}

type Message struct {
	Addr      string
	TargetSeg int
	KillWorm  bool
}

func InitComm(HostName, hostPort, wormgatePort string, get func() int, set func(int), leader func() string) *Comm {
	c := &Comm{
		HostName:     HostName,
		WormgatePort: wormgatePort,
		HostPort:     hostPort,
		client:       &http.Client{},
		getTarget:    get,
		setTarget:    set,
		getLeader:    leader,
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
		TargetSeg: c.getTarget(),
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
		//reader := bytes.NewReader(data)
		go c.udpPing(addr, ch, data)
	}

	for i := 0; i < len(hosts); i++ {
		host := <-ch
		if host != "" {
			activeHosts = append(activeHosts, host)
		}
	}
	return activeHosts
}

func (c Comm) doPing(addr string, ch chan<- string, body *bytes.Reader) {
	url := fmt.Sprintf("http://%s%s/alive", addr, c.HostPort)

	resp, err := c.client.Post(url, "application/json", body)
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

			reader := bytes.NewReader(data)

			msg := &Message{}

			err := json.NewDecoder(reader).Decode(msg)
			if err != nil {
				fmt.Println(err)
			}

			if msg.Addr == c.getLeader() || msg.TargetSeg == 0 {
				c.setTarget(msg.TargetSeg)
			}

			_, err = conn.WriteTo([]byte("alive"), addr)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (c Comm) udpPing(addr string, ch chan<- string, data []byte) {

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
	_, err = conn.Write(data)
	if err != nil {
		ch <- ""
		return
	}

	response := make([]byte, 128)

	t := time.Now()

	conn.SetReadDeadline(t.Add(time.Millisecond * 10))

	bytes, err := conn.Read(response)
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
