package main

import (
	"flag"
	"fmt"
	"github.com/joonnna/worm/communication"
	"github.com/joonnna/worm/util"
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
	targetSegments int
	currentLeader  string
	ownHash        big.Int
	hostMap        map[string]big.Int
	logFile        *os.File
	*Logger

	updateTime int64

	leaderFile *os.File
	spreadFile string

	targetMutex sync.RWMutex
	leaderMutex sync.RWMutex

	*communication.Comm
}

func (s *Seg) StartSegmentServer(segPort string) {

	runtime.GOMAXPROCS(runtime.NumCPU())

	http.HandleFunc("/", s.indexHandler)
	http.HandleFunc("/targetsegments", s.targetSegmentsHandler)
	http.HandleFunc("/shutdown", s.shutdownHandler)

	http.HandleFunc("/alive", s.aliveHandler)
	http.HandleFunc("/updatetarget", s.updateSegmentsHandler)

	startup := make(chan bool)

	go s.logLeader()
	//go s.InitUdp()
	go s.CommStatus(startup)

	<-startup

	go s.monitorWorm()
	//go s.initUdp()

	//go s.initTcp()

	err := http.ListenAndServe(segPort, nil)
	if err != nil {
		s.Err.Panic(err)
	}
}

func (s *Seg) logLeader() {
	for {
		time.Sleep(time.Second * 10)
		//str := fmt.Sprintf("%s : %d : %s\n", s.HostName, s.getTargetSegments(), s.getLeader())
		str := fmt.Sprintf("%s : %d : %s\n", s.HostName, s.getTargetSegments(), s.getLeader())
		s.leaderFile.Write([]byte(str))
		//str := strings.Join(s.GetActiveHosts(), " ")
		//s.leaderFile.Write([]byte(fmt.Sprintf("%s : %s\n", s.HostName, str)))
		s.leaderFile.Sync()
	}
}

func (s *Seg) monitorWorm() {
	//	prevActive := s.GetActiveHosts()
	//	s.updateMap(prevActive)

	for {
		activeSegs := s.GetActiveHosts()
		/*
			diff := util.SliceDiff(prevActive, activeSegs)

			if len(diff) > 0 {
				s.Info.Printf("changed active list : %d\n", len(diff))
				s.updateMap(diff)
			}
		*/
		s.updateMap(activeSegs)

		if len(activeSegs) == 0 {
			s.setLeader(s.HostName)
		}

		//if s.getLeader() == s.HostName {

		if s.spreadFile == "" {
			s.tarFile()
			defer os.Remove(s.spreadFile)
		}

		//s.Info.Println("IM THE BIGGEST MOFO")

		leader := s.getLeader()

		for i, host := range activeSegs {
			if host == leader {
				activeSegs = append(activeSegs[:i], activeSegs[i+1:]...)
			}
		}

		s.checkTarget((len(activeSegs) + 1), activeSegs)
		//time.Sleep(time.Second * 1)
		//}

		//	prevActive = activeSegs
	}
}

func (s *Seg) updateMap(activeSegs []string) {

	highestHash := *big.NewInt(0)

	for _, host := range activeSegs {

		newHash := *big.NewInt(0)

		if hash, ok := s.hostMap[host]; !ok {
			newHash = util.ComputeHash(host)
			s.hostMap[host] = newHash
		} else {
			newHash = hash
		}

		if util.CmpHash(newHash, highestHash) == 1 {
			highestHash = newHash
			s.setLeader(host)
		}
	}

	if util.CmpHash(s.ownHash, highestHash) == 1 {
		s.setLeader(s.HostName)
	}

}

func (s *Seg) checkTarget(numSegs int, activeHosts []string) {

	allHosts := s.GetAllHosts()
	target := s.getTargetSegments()

	//s.Info.Printf("There is %d segments alive, should be: %d", numSegs, target)

	inactiveHosts := util.SliceDiff(activeHosts, allHosts)

	if target > numSegs {
		s.addSegments((target - numSegs), inactiveHosts)
	} else if target < numSegs {
		s.removeSegments((numSegs - target), activeHosts)
	}
}

func (s Seg) killSegment(address string, ch chan<- string) {
	url := fmt.Sprintf("http://%s%s/shutdown", address, s.HostPort)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s.Err.Println("GET error ", err)
		ch <- address
		return
	}

	err = s.ContactHostHttp(req)
	if err != nil {
		ch <- ""
		return
	}
	ch <- ""
}

func (s Seg) removeSegments(numSegs int, hosts []string) {
	ch := make(chan string, numSegs)

	var failed []string

	for _, host := range hosts[:numSegs] {
		s.Info.Printf("Killing %s", host)
		go s.killSegment(host, ch)
	}

	for i := 0; i < numSegs; i++ {
		val := <-ch
		if val != "" {
			failed = append(failed, val)
		}
	}

	rest := len(failed)

	if rest > 0 {
		diff := util.SliceDiff(failed, hosts)
		s.removeSegments(rest, diff)
	}
}

func (s Seg) addSegments(numSegs int, hosts []string) {
	ch := make(chan string, numSegs)

	var failed []string

	for _, host := range hosts[:numSegs] {
		s.Info.Printf("Spreading to %s", host)
		go s.sendSegment(host, ch)
	}

	for i := 0; i < numSegs; i++ {
		val := <-ch
		if val != "" {
			failed = append(failed, val)
		}
	}

	rest := len(failed)
	if rest > 0 {
		diff := util.SliceDiff(failed, hosts)
		s.addSegments(rest, diff)
	}

}

func (s Seg) sendSegment(address string, ch chan<- string) {

	url := fmt.Sprintf("http://%s%s/wormgate?sp=%s", address, s.WormgatePort, s.HostPort)

	file, err := os.Open(s.spreadFile)
	if err != nil {
		s.Err.Printf("Could not read input file %s", err)
		ch <- address
		return
	}
	defer file.Close()

	req, err := http.NewRequest("POST", url, file)
	if err != nil {
		s.Err.Println("POST error ", err)
		ch <- address
		return
	}

	req.Header.Set("targetsegment", strconv.Itoa(s.getTargetSegments()))

	err = s.ContactHostHttp(req)
	if err != nil {
		s.Err.Println(err)
		ch <- address
		return
	}
	ch <- ""
}

func (s Seg) sendTargetSegment(address string, ts int, ch chan<- bool) {
	url := fmt.Sprintf("http://%s%s/updatetarget", address, s.HostPort)
	body := strings.NewReader(fmt.Sprint(ts))

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		s.Err.Println(err)
		ch <- false
		return
	}

	req.Header.Set("targetsegment", strconv.Itoa(ts))

	err = s.ContactHostHttp(req)
	if err != nil {
		s.Err.Println(err)
	}

	ch <- true
}

func (s *Seg) tarFile() {

	filename := "tmp.tar.gz"

	gopath := os.Getenv("GOPATH")

	err := os.Chdir(gopath + "/bin")
	if err != nil {
		s.Err.Panic(err)
	}

	tarCmd := exec.Command("tar", "-zc", "-f", filename, "segment")
	err = tarCmd.Run()
	if err != nil {
		s.Err.Println(err)
	}

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

func addFlags(flagset *flag.FlagSet, wormPort, segPort, mode, host *string, target *int) {
	flagset.StringVar(wormPort, "wp", ":8181", "wormgate port (prefix with colon)")
	flagset.StringVar(segPort, "sp", ":8182", "segment port (prefix with colon)")
	flagset.StringVar(mode, "mode", "run", "segment mode")
	flagset.StringVar(host, "host", "compute-1-0", "host to spread to")
	flagset.IntVar(target, "target", 2, "segment target number")
}

func main() {

	var hostName, segPort, wormPort, spreadHost, mode string
	var targetSegments int

	host, _ := os.Hostname()

	hostName = strings.Split(host, ".")[0]

	args := flag.NewFlagSet("args", flag.ExitOnError)
	addFlags(args, &wormPort, &segPort, &mode, &spreadHost, &targetSegments)

	args.Parse(os.Args[1:])

	errPrefix := fmt.Sprintf("\x1b[31m %s \x1b[0m", hostName)
	infoPrefix := fmt.Sprintf("\x1b[32m %s \x1b[0m", hostName)

	logFile, _ := os.OpenFile("/home/jmi021/log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

	leaderFile, _ := os.OpenFile("/home/jmi021/leader", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

	errLog := log.New(logFile, errPrefix, log.Lshortfile)
	infoLog := log.New(logFile, infoPrefix, log.Lshortfile)

	s := &Seg{
		targetSegments: targetSegments,
		Logger:         &Logger{Err: errLog, Info: infoLog},
		logFile:        logFile,
		ownHash:        util.ComputeHash(hostName),
		hostMap:        make(map[string]big.Int),
		leaderFile:     leaderFile,
	}

	comm := communication.InitComm(hostName, segPort, wormPort, s.getTargetSegments)
	s.Comm = comm

	switch mode {

	case "spread":
		ch := make(chan string)
		s.tarFile()
		s.sendSegment(spreadHost, ch)
		<-ch
		os.Remove(s.spreadFile)
	case "run":
		s.Info.Println("FUCK THE FUCK YE WE ARE STARTED")
		s.StartSegmentServer(segPort)

	default:
		s.Err.Fatalf("Unknown mode %q\n", os.Args[1])
	}
}
