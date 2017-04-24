package segment

import (
	"fmt"
	"github.com/joonnna/worm/util"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	Err  *log.Logger
	Info *log.Logger
}

type Seg struct {
	hostName       string
	segmentPort    string
	wormgatePort   string
	targetSegments int
	currentLeader  string

	client *http.Client

	logFile *os.File
	*Logger

	allHosts    []string
	activeHosts []string

	spreadFile string

	targetMutex      sync.RWMutex
	leaderMutex      sync.RWMutex
	allHostsMutex    sync.RWMutex
	activeHostsMutex sync.RWMutex
}

func (s *Seg) StartSegmentServer() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	http.HandleFunc("/", s.IndexHandler)
	http.HandleFunc("/targetsegments", s.targetSegmentsHandler)
	http.HandleFunc("/shutdown", s.shutdownHandler)
	http.HandleFunc("/alive", s.aliveHandler)
	http.HandleFunc("/updatetarget", s.updateSegmentsHandler)

	s.Info.Printf("Starting Segment server on %s%s\n", s.hostName, s.segmentPort)

	go s.wormStatus()
	go s.monitorWorm()

	err := http.ListenAndServe(s.segmentPort, nil)
	if err != nil {
		s.Err.Panic(err)
	}
}

func (s Seg) IndexHandler(w http.ResponseWriter, r *http.Request) {

	// We don't use the request body. But we should consume it anyway.
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	killRateGuess := 2.0

	fmt.Fprintf(w, "%.3f\n", killRateGuess)
}

func (s *Seg) targetSegmentsHandler(w http.ResponseWriter, r *http.Request) {
	var ts int32
	pc, rateErr := fmt.Fscanf(r.Body, "%d", &ts)
	if pc != 1 || rateErr != nil {
		s.Err.Printf("Error parsing targetSegments (%d items): %s", pc, rateErr)
	}

	// Consume and close rest of body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	s.Info.Printf("New targetSegments: %d", ts)

	s.setTargetSegments(int(ts))

	activeHosts := s.getActiveHosts()

	leader := s.getLeader()
	if leader != s.hostName {
		s.sendTargetSegment(leader, int(ts))
	}

	for _, host := range activeHosts {
		if host == leader {
			continue
		}

		s.sendTargetSegment(host, int(ts))
	}
}

func (s *Seg) updateSegmentsHandler(w http.ResponseWriter, r *http.Request) {
	var ts int32
	pc, rateErr := fmt.Fscanf(r.Body, "%d", &ts)
	if pc != 1 || rateErr != nil {
		s.Err.Printf("Error parsing targetSegments (%d items): %s", pc, rateErr)
	}

	// Consume and close rest of body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	s.Info.Printf("New targetSegments: %d", ts)

	s.setTargetSegments(int(ts))

	leader := s.getLeader()

	if leader == s.hostName {
		activeSegs := s.getActiveHosts()

		for _, host := range activeSegs {
			s.sendTargetSegment(host, int(ts))
		}
	}
}

func (s Seg) shutdownHandler(w http.ResponseWriter, r *http.Request) {

	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	// Shut down
	s.Info.Printf("Received shutdown command, committing suicide")
	os.Exit(0)
}

func (s Seg) aliveHandler(w http.ResponseWriter, r *http.Request) {
	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
}

func (s *Seg) monitorWorm() {
	hostMap := make(map[string]big.Int)
	for {

		time.Sleep(time.Second)
		highestHash := *big.NewInt(0)

		allHosts := s.getAllHosts()
		activeSegs := s.getActiveHosts()

		for _, host := range activeSegs {

			hash := *big.NewInt(0)

			if hash, ok := hostMap[host]; !ok {
				hash = util.ComputeHash(host)
				hostMap[host] = hash
			}

			if util.CmpHash(hash, highestHash) == 1 {
				highestHash = hash
				s.setLeader(host)
			}
		}

		if len(activeSegs) == 0 {
			s.setLeader(s.hostName)
		}

		if s.getLeader() == s.hostName {

			if s.spreadFile == "" {
				s.tarFile()
				defer os.Remove(s.spreadFile)
			}

			s.Info.Println("IM THE BIGGEST MOFO")
			s.checkTarget((len(activeSegs) + 1), allHosts, activeSegs)
		}
	}
}

func (s *Seg) wormStatus() {
	for {
		//time.Sleep(time.Second * 1)
		allHosts := util.FetchReachableHosts(s.wormgatePort, s.hostName)
		s.setAllHosts(allHosts)
		s.setActiveHosts(s.checkActiveSegs(allHosts))
	}
}

func (s Seg) pingHost(address string) bool {
	url := fmt.Sprintf("http://%s%s/alive", address, s.segmentPort)

	resp, err := s.client.Get(url)
	if err != nil {
		return false
	}

	if resp.StatusCode == 200 || resp.StatusCode == 409 {
		return true
	}

	return false
}

func (s *Seg) checkTarget(numSegs int, allHosts []string, activeHosts []string) {

	target := s.getTargetSegments()

	s.Info.Printf("There is %d segments alive, should be: %d", numSegs, target)

	inactiveHosts := util.SliceDiff(activeHosts, allHosts)

	if target > numSegs {
		s.addSegments((target - numSegs), inactiveHosts)
	} else if target < numSegs {
		s.removeSegments((numSegs - target), activeHosts)
	}
}

func (s Seg) removeSegments(numSegs int, hosts []string) {
	for _, host := range hosts[:numSegs] {
		s.Info.Printf("Killing %s", host)
		err := killSegment((host + s.segmentPort))
		if err != nil {
			s.Err.Println(err)
		}
	}
}

func killSegment(address string) error {
	_, err := http.Get(fmt.Sprintf("http://%s/shutdown", address))
	if err != nil {
		return err
	}

	return nil
}

func (s Seg) addSegments(numSegs int, hosts []string) {
	var counter int

	for _, host := range hosts {

		s.Info.Printf("Spreading to %s", host)
		err := s.sendSegment(host)
		if err == nil {
			counter += 1
		}

		if counter == numSegs {
			break
		}
	}
}

func (s Seg) checkActiveSegs(allHosts []string) []string {

	var hosts []string

	for _, host := range allHosts {
		if ok := s.pingHost(host); ok {
			hosts = append(hosts, host)
		}
	}

	return hosts
}

func (s Seg) sendSegment(address string) error {

	url := fmt.Sprintf("http://%s%s/wormgate?sp=%s", address, s.wormgatePort, s.segmentPort)

	/*
		filename := "tmp.tar.gz"

		// ship the binary and the qml file that describes our screen output
		tarCmd := exec.Command("tar", "-zc", "-f", filename, "main")
		tarCmd.Run()

		file, err := os.Open(filename)
		if err != nil {
			s.Err.Panic("Could not read input file", err)
		}
		defer os.Remove(filename)
	*/

	file, err := os.Open(s.spreadFile)
	if err != nil {
		s.Err.Panic("Could not read input file", err)
	}
	defer file.Close()

	req, err := http.NewRequest("POST", url, file)
	if err != nil {
		s.Err.Println("POST error ", err)
		return err
	}

	req.Header.Set("targetsegment", strconv.Itoa(s.targetSegments))

	resp, err := s.client.Do(req)
	if err != nil {
		s.Err.Println("POST error ", err)
	} else {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
	/*
		if resp.StatusCode == 200 {
			s.Info.Println("Received OK from server")
		} else {
			s.Err.Println("Response: ", resp)
		}
	*/
	return err
}

func (s Seg) sendTargetSegment(address string, ts int) {
	url := fmt.Sprintf("http://%s%s/updatetarget", address, s.segmentPort)
	body := strings.NewReader(fmt.Sprint(ts))

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		s.Err.Println(err)
		return
	}

	req.Header.Set("targetsegment", strconv.Itoa(ts))

	resp, err := s.client.Do(req)
	if err != nil {
		s.Err.Println(err)
	} else {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}

}

func (s *Seg) tarFile() {

	filename := "tmp.tar.gz"

	// ship the binary and the qml file that describes our screen output
	tarCmd := exec.Command("tar", "-zc", "-f", filename, "main")
	err := tarCmd.Run()
	if err != nil {
		s.Err.Println(err)
	}
	/*
		file, err := os.Open(filename)
		if err != nil {
			log.Panic("Could not read input file", err)
		}
	*/
	s.spreadFile = filename
}

func (s *Seg) setTargetSegments(target int) {
	s.targetMutex.Lock()
	s.targetSegments = target
	s.targetMutex.Unlock()
}

func (s *Seg) getTargetSegments() int {
	s.targetMutex.RLock()
	ret := s.targetSegments
	s.targetMutex.RUnlock()

	return ret
}

func (s *Seg) setLeader(host string) {
	s.leaderMutex.Lock()
	s.currentLeader = host
	s.leaderMutex.Unlock()
}

func (s *Seg) getLeader() string {
	s.leaderMutex.RLock()
	ret := s.currentLeader
	s.leaderMutex.RUnlock()

	return ret
}

func (s *Seg) setAllHosts(hosts []string) {
	s.allHostsMutex.Lock()
	s.allHosts = hosts
	s.allHostsMutex.Unlock()
}

func (s *Seg) getAllHosts() []string {
	s.allHostsMutex.RLock()
	ret := s.allHosts
	s.allHostsMutex.RUnlock()

	return ret
}

func (s *Seg) setActiveHosts(hosts []string) {
	s.activeHostsMutex.Lock()
	s.activeHosts = hosts
	s.activeHostsMutex.Unlock()
}

func (s *Seg) getActiveHosts() []string {
	s.activeHostsMutex.RLock()
	ret := s.activeHosts
	s.activeHostsMutex.RUnlock()

	return ret
}

func Run(wormPort, segPort, mode, spreadHost string, targetSegments int) {

	host, _ := os.Hostname()

	hostName := strings.Split(host, ".")[0]

	errPrefix := fmt.Sprintf("\x1b[31m %s \x1b[0m", hostName)
	infoPrefix := fmt.Sprintf("\x1b[32m %s \x1b[0m", hostName)

	logFile, _ := os.OpenFile("/home/jmi021/log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

	errLog := log.New(logFile, errPrefix, log.Lshortfile)
	infoLog := log.New(logFile, infoPrefix, log.Lshortfile)

	s := &Seg{
		hostName:       hostName,
		segmentPort:    segPort,
		wormgatePort:   wormPort,
		targetSegments: targetSegments,
		client:         &http.Client{},
		Logger:         &Logger{Err: errLog, Info: infoLog},
		logFile:        logFile,
	}

	switch mode {

	case "spread":
		s.tarFile()
		s.sendSegment(spreadHost)
		os.Remove(s.spreadFile)
	case "run":
		s.Info.Println("FUCK THE FUCK YE WE ARE STARTED")
		s.StartSegmentServer()

	default:
		s.Err.Fatalf("Unknown mode %q\n", os.Args[1])
	}
}
