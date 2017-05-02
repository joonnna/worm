package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

type Message struct {
	Addr      string
	TargetSeg int
}

func (s *Seg) listenUDP() {

	conn, err := net.ListenPacket("udp", s.HostName+":19032")
	if err != nil {
		os.Exit(1)
	}
	defer conn.Close()

	s.udpConn = conn

	for {

		data := make([]byte, 300)
		_, addr, err := conn.ReadFrom(data)
		if err != nil {
			s.Err.Println(err)
		} else {

			reader := bytes.NewReader(data)

			msg := &Message{}

			err := json.NewDecoder(reader).Decode(msg)
			if err != nil {
				s.Err.Println(err)
			}

			if msg.Addr == s.getLeader() || msg.TargetSeg == 0 {
				s.setTargetSegments(msg.TargetSeg)
			}

			_, _ = conn.WriteTo([]byte("alive"), addr)
		}
	}
}

func (s *Seg) PingHosts() []string {
	hosts := s.GetAllHosts()

	var activeHosts []string

	ch := make(chan string, len(hosts))

	msg := &Message{
		Addr:      s.HostName,
		TargetSeg: s.getTargetSegments(),
	}

	marsh, err := json.Marshal(msg)
	if err != nil {
		s.Err.Println(err)
	}

	buf := bytes.NewBuffer(marsh)

	err = json.NewEncoder(buf).Encode(msg)
	if err != nil {
		s.Err.Println(err)
	}

	data := buf.Bytes()

	for _, addr := range hosts {
		go s.udpPing(addr, ch, data)
	}

	for i := 0; i < len(hosts); i++ {
		host := <-ch
		if host != "" {
			activeHosts = append(activeHosts, host)
		}
	}
	return activeHosts
}

func (s Seg) httpPing(addr string, ch chan<- string, body *bytes.Reader) {
	url := fmt.Sprintf("http://%s%s/alive", addr, s.HostPort)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		ch <- ""
		return
	}

	err = s.ContactHostHttp(req)
	if err != nil {
		ch <- ""
		return
	}

	ch <- ""
}

func (s Seg) udpPing(addr string, ch chan<- string, data []byte) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr+":19032")
	if err != nil {
		ch <- ""
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
