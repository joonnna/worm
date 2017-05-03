package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

func (s *Seg) indexHandler(w http.ResponseWriter, r *http.Request) {
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	killRateGuess := s.getKillRate()

	fmt.Fprintf(w, "%.3f\n", killRateGuess)
}

func (s *Seg) targetSegmentsHandler(w http.ResponseWriter, r *http.Request) {
	var ts int32

	defer r.Body.Close()

	if s.getTargetSegments() == 0 {
		io.Copy(ioutil.Discard, r.Body)
		return
	}

	pc, rateErr := fmt.Fscanf(r.Body, "%d", &ts)
	if pc != 1 || rateErr != nil {
		s.Err.Printf("Error parsing targetSegments (%d items): %s", pc, rateErr)
		return
	}
	io.Copy(ioutil.Discard, r.Body)

	s.setTargetSegments(int(ts))

	leader := s.getLeader()

	if s.HostName != leader {
		ch := make(chan bool, 1)
		s.sendTargetSegment(leader, int(ts), ch)
		<-ch
	}

}

func (s *Seg) updateSegmentsHandler(w http.ResponseWriter, r *http.Request) {
	var ts int32

	defer r.Body.Close()

	if s.getTargetSegments() == 0 {
		io.Copy(ioutil.Discard, r.Body)
		return
	}

	pc, rateErr := fmt.Fscanf(r.Body, "%d", &ts)
	if pc != 1 || rateErr != nil {
		s.Err.Printf("Error parsing targetSegments (%d items): %s", pc, rateErr)
		return
	}

	s.setTargetSegments(int(ts))

}

func (s *Seg) shutdownHandler(w http.ResponseWriter, r *http.Request) {
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	leader := s.getLeader()

	ch := make(chan bool, 1)

	s.sendTargetSegment(leader, 0, ch)
	<-ch

}

func (s *Seg) suicideHandler(w http.ResponseWriter, r *http.Request) {
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	os.Remove(s.spreadFileName)

	s.udpConn.Close()

	s.httpListener.Close()

	os.Exit(0)
}

func (s *Seg) aliveHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	msg := &Message{}

	err := json.NewDecoder(r.Body).Decode(msg)
	if err != nil {
		s.Err.Println(err)
	}

	if msg.Addr == s.getLeader() || msg.TargetSeg == 0 {
		s.setTargetSegments(msg.TargetSeg)
	}
}
