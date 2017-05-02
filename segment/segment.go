package main

import (
	"bytes"

	"flag"
	"fmt"
	"github.com/joonnna/worm/communication"
	"github.com/joonnna/worm/util"
	"io/ioutil"
	"log"
	"math/big"
	"net"
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

	ownHash big.Int

	leaderFile *os.File
	logFile    *os.File
	*Logger

	udpConn      net.PacketConn
	httpListener net.Listener

	killRate float32

	spreadFile     []byte
	spreadFileName string

	hostMap map[string]big.Int

	numAddedMutex sync.RWMutex
	addMapMutex   sync.RWMutex
	targetMutex   sync.RWMutex
	leaderMutex   sync.RWMutex
	killRateMutex sync.RWMutex

	*communication.Comm
}

func (s *Seg) StartSegmentServer(segPort string) {

	runtime.GOMAXPROCS(runtime.NumCPU())
	//http.DefaultTransport.(*http.Transport).MaxIdleConns = 1000

	http.HandleFunc("/", s.indexHandler)
	http.HandleFunc("/targetsegments", s.targetSegmentsHandler)
	http.HandleFunc("/shutdown", s.shutdownHandler)
	http.HandleFunc("/suicide", s.suicideHandler)

	http.HandleFunc("/updatetarget", s.updateSegmentsHandler)

	startup := make(chan bool)

	go s.listenUDP()
	//go s.logLeader()
	go s.CommStatus(startup)

	<-startup

	go s.monitorWorm()

	l, err := net.Listen("tcp", s.HostPort)
	if err != nil {
		s.Err.Panic(err)
	}

	s.httpListener = l

	err = http.Serve(l, nil)
	if err != nil {
		s.Err.Panic(err)
	}
}

func (s *Seg) logLeader() {
	for {
		time.Sleep(time.Second * 10)
		//str := fmt.Sprintf("%s : %d : %s\n", s.HostName, s.getTargetSegments(), s.getLeader())
		str := fmt.Sprintf("%s : %d : %s : %d : %f\n", s.HostName, s.getTargetSegments(), s.getLeader(), len(s.GetActiveHosts()), s.getKillRate())
		s.leaderFile.Write([]byte(str))
		//str := strings.Join(s.GetActiveHosts(), " ")
		//s.leaderFile.Write([]byte(fmt.Sprintf("%s : %s\n", s.HostName, str)))
		s.leaderFile.Sync()
	}
}

func (s *Seg) monitorWorm() {

	go s.estimateKillRate()

	for {

		activeSegs := s.GetActiveHosts()

		s.updateMap(activeSegs)

		numSegs := len(activeSegs)

		if numSegs == 0 {
			s.setLeader(s.HostName)
		}

		leader := s.getLeader()

		target := s.getTargetSegments()

		if s.HostName == leader && target == 0 {
			s.killWorm()
		}

		s.checkTarget((numSegs + 1), activeSegs)
	}
}

func (s *Seg) killWorm() {

	s.Info.Println("Killing entire worm")

	for {
		activeSegs := s.GetActiveHosts()

		numSegs := len(activeSegs)

		if numSegs == 0 {
			s.Info.Println("Worm Dead committing suicide")

			os.Remove(s.spreadFileName)
			s.httpListener.Close()

			os.Exit(0)
		}
		s.removeSegments(numSegs, activeSegs)
	}

}

func (s *Seg) updateMap(activeSegs []string) {

	highestHash := *big.NewInt(0)
	var leaderHost string

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
			leaderHost = host
		}
	}

	if util.CmpHash(s.ownHash, highestHash) == 1 {
		s.setLeader(s.HostName)
	} else {
		s.setLeader(leaderHost)
	}
}

func (s *Seg) checkTarget(numSegs int, activeHosts []string) {

	leader := s.getLeader()
	allHosts := s.GetAllHosts()
	target := s.getTargetSegments()

	inactiveHosts := util.SliceDiff(activeHosts, allHosts)

	availableNodes := len(inactiveHosts)

	newTarget := target - numSegs

	if target > numSegs {
		if newTarget-availableNodes > 0 {
			s.addSegments(availableNodes, inactiveHosts)
		} else {
			s.addSegments(newTarget, inactiveHosts)
		}

	} else if target < numSegs && leader == s.HostName {
		s.removeSegments((numSegs - target), activeHosts)
	}
}

func (s Seg) killSegment(address string, ch chan<- string) {
	url := fmt.Sprintf("http://%s%s/suicide", address, s.HostPort)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
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

func (s *Seg) sendSegment(address string, ch chan<- string) {
	url := fmt.Sprintf("http://%s%s/wormgate?sp=%s", address, s.WormgatePort, s.HostPort)

	req, err := http.NewRequest("POST", url, bytes.NewReader(s.spreadFile))
	if err != nil {
		ch <- address
		return
	}

	req.Header.Set("targetsegment", strconv.Itoa(s.getTargetSegments()))

	err = s.ContactHostHttp(req)
	if err != nil {
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

func (s *Seg) estimateKillRate() {

	active := s.GetActiveHosts()
	prev := len(active)

	time.Sleep(time.Second)

	for {

		active := s.GetActiveHosts()
		curr := len(active)

		diff := prev - curr

		if diff > 0 {
			killRate := float32(diff)
			s.setKillRate(killRate)
		}

		prev = curr

		time.Sleep(time.Second)
	}
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
		s.Err.Panic(err)
	}

	file, err := os.Open(filename)
	if err != nil {
		s.Err.Panic(err)
	}

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		s.Err.Panic(err)
	}

	s.spreadFile = bytes
	s.spreadFileName = file.Name()

	file.Close()
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
		leaderFile:     leaderFile,
		hostMap:        make(map[string]big.Int),
	}

	comm := communication.InitComm(hostName, segPort, wormPort, s)
	s.Comm = comm

	s.tarFile()

	defer os.Remove(s.spreadFileName)

	switch mode {

	case "spread":
		ch := make(chan string)
		s.sendSegment(spreadHost, ch)
		<-ch
	case "run":
		s.StartSegmentServer(segPort)
	default:
		s.Err.Fatalf("Unknown mode %q\n", os.Args[1])
	}
}
