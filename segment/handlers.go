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

func (s Seg) indexHandler(w http.ResponseWriter, r *http.Request) {

	// We don't use the request body. But we should consume it anyway.
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	killRateGuess := 2.0

	fmt.Fprintf(w, "%.3f\n", killRateGuess)
}

func (s *Seg) targetSegmentsHandler(w http.ResponseWriter, r *http.Request) {

	s.Info.Println("KA I FAEN")
	var ts int32
	pc, rateErr := fmt.Fscanf(r.Body, "%d", &ts)
	if pc != 1 || rateErr != nil {
		s.Err.Printf("Error parsing targetSegments (%d items): %s", pc, rateErr)
		return
	}

	// Consume and close rest of body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	s.Info.Printf("TargetHandler targetSegments: %d", ts)

	s.setTargetSegments(int(ts))

	leader := s.getLeader()

	ch := make(chan bool, 1)

	s.sendTargetSegment(leader, int(ts), ch)
	<-ch

	/*
		activeSegs := s.GetActiveHosts()

		ch := make(chan bool, len(activeSegs))

		leader := s.getLeader()
		s.Info.Printf("YOYOYOYOYOY :%s\n", leader)

		if leader != s.HostName {
			s.sendTargetSegment(leader, int(ts), ch)
			<-ch
		} else {
			for _, host := range activeSegs {
				go s.sendTargetSegment(host, int(ts), ch)
			}

			for i := 0; i < len(activeSegs); i++ {
				<-ch
			}
		}
	*/
}

func (s *Seg) updateSegmentsHandler(w http.ResponseWriter, r *http.Request) {

	var ts int32
	pc, rateErr := fmt.Fscanf(r.Body, "%d", &ts)
	if pc != 1 || rateErr != nil {
		s.Err.Printf("Error parsing targetSegments (%d items): %s", pc, rateErr)
		return
	}

	// Consume and close rest of body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	s.Info.Printf("New targetSegments: %d", ts)

	s.setTargetSegments(int(ts))
	/*
		leader := s.getLeader()

		if leader == s.HostName {
			activeSegs := s.GetActiveHosts()

			ch := make(chan bool, len(activeSegs))

			for _, host := range activeSegs {
				go s.sendTargetSegment(host, int(ts), ch)
			}

			for i := 0; i < len(activeSegs); i++ {
				<-ch
			}
		}
	*/
}

func (s *Seg) shutdownHandler(w http.ResponseWriter, r *http.Request) {

	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	// Shut down
	s.Info.Printf("Received shutdown command, committing suicide")

	//s.UdpConn.Close()
	os.Exit(0)
}

func (s *Seg) aliveHandler(w http.ResponseWriter, r *http.Request) {

	defer r.Body.Close()
	msg := &communication.Message{}

	err := json.NewDecoder(r.Body).Decode(msg)
	if err != nil {
		s.Err.Println(err)
	}

	if msg.Addr == s.getLeader() {
		s.setTargetSegments(msg.TargetSeg)
	}
}
