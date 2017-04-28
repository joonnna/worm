package main

import (
	"encoding/json"
	"fmt"
	"github.com/joonnna/worm/communication"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

func (s *Seg) indexHandler(w http.ResponseWriter, r *http.Request) {

	// We don't use the request body. But we should consume it anyway.
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	killRateGuess := s.GetKillRate()

	fmt.Fprintf(w, "%.3f\n", killRateGuess)
}

func (s *Seg) targetSegmentsHandler(w http.ResponseWriter, r *http.Request) {
	var ts int32

	defer r.Body.Close()

	if s.GetTargetSegments() == 0 {
		io.Copy(ioutil.Discard, r.Body)
		return
	}

	pc, rateErr := fmt.Fscanf(r.Body, "%d", &ts)
	if pc != 1 || rateErr != nil {
		s.Err.Printf("Error parsing targetSegments (%d items): %s", pc, rateErr)
		return
	}
	// Consume and close rest of body
	io.Copy(ioutil.Discard, r.Body)

	s.Info.Printf("TargetHandler targetSegments: %d", ts)

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

	if s.GetTargetSegments() == 0 {
		io.Copy(ioutil.Discard, r.Body)
		return
	}

	pc, rateErr := fmt.Fscanf(r.Body, "%d", &ts)
	if pc != 1 || rateErr != nil {
		s.Err.Printf("Error parsing targetSegments (%d items): %s", pc, rateErr)
		return
	}

	s.setTargetSegments(int(ts))

	s.Info.Printf("New targetSegments: %d", ts)
	// Consume and close rest of body

}

func (s *Seg) shutdownHandler(w http.ResponseWriter, r *http.Request) {

	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	leader := s.getLeader()

	ch := make(chan bool, 1)

	s.sendTargetSegment(leader, 0, ch)
	<-ch

	// Shut down
	s.Info.Printf("Received shutdown command, KILL WORM")
	//s.UdpConn.Close()
}

func (s *Seg) suicideHandler(w http.ResponseWriter, r *http.Request) {

	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	// Shut down
	s.Info.Printf("Received suicide command, committing suicide")

	//s.UdpConn.Close()

	os.Remove(s.spreadFileName)

	s.httpListener.Close()

	os.Exit(0)
}

func (s *Seg) aliveHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	msg := &communication.Message{}

	err := json.NewDecoder(r.Body).Decode(msg)
	if err != nil {
		s.Err.Println(err)
	}

	if msg.Addr == s.getLeader() || msg.TargetSeg == 0 {
		s.setTargetSegments(msg.TargetSeg)
	}
}
