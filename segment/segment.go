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
	"strings"
	"sync"
)

type Seg struct {
	hostName       string
	segmentPort    string
	wormgatePort   string
	targetSegments int

	targetMutex *sync.Mutex
}

func (s Seg) StartSegmentServer() {

	http.HandleFunc("/", s.IndexHandler)
	http.HandleFunc("/targetSegments", s.targetSegmentsHandler)
	http.HandleFunc("/shutdown", s.shutdownHandler)
	http.HandleFunc("/alive", s.aliveHandler)

	log.Printf("Starting Segment server on %s%s\n", s.hostName, s.segmentPort)
	log.Printf("Reachable hosts: %s", strings.Join(util.FetchReachableHosts(s.wormgatePort), " "))

	go s.monitorWorm()

	err := http.ListenAndServe(s.segmentPort, nil)
	if err != nil {
		log.Panic(err)
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
		log.Printf("Error parsing targetSegments (%d items): %s", pc, rateErr)
	}

	// Consume and close rest of body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	log.Printf("New targetSegments: %d", ts)

	s.targetMutex.Lock()
	s.targetSegments = int(ts)
	s.targetMutex.Unlock()
}

func (s Seg) shutdownHandler(w http.ResponseWriter, r *http.Request) {

	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()

	// Shut down
	log.Printf("Received shutdown command, committing suicide")
	os.Exit(0)
}

func (s Seg) aliveHandler(w http.ResponseWriter, r *http.Request) {
	// Consume and close body
	io.Copy(ioutil.Discard, r.Body)
	r.Body.Close()
}

func (s Seg) monitorWorm() {
	hostMap := make(map[string]*big.Int)

	for {
		var counter int
		highestHash := big.NewInt(0)

		allHosts := util.FetchReachableHosts(s.wormgatePort)

		for _, host := range allHosts {

			hash := big.NewInt(0)

			if hash, ok := hostMap[host]; !ok {
				hash = util.ComputeHash(host)
				hostMap[host] = hash
			}

			if ok := pingHost(host); ok {
				counter += 1
				if util.CmpHash(hash, highestHash) {
					highestHash = hash
				}
			}
		}

		ownHash := hostMap[s.hostName]
		if util.CmpHash(ownHash, highestHash) {
			s.checkTarget(counter, allHosts)
		}
	}
}

func pingHost(address string) bool {
	resp, err := http.Get(fmt.Sprintf("http://%s/alive", address))
	if err != nil {
		log.Fatal(err)
	}

	if resp.Status != "200" {
		return false
	}

	return true
}

func (s Seg) checkTarget(numSegs int, allHosts []string) {
	var target int

	s.targetMutex.Lock()
	target = s.targetSegments
	s.targetMutex.Unlock()

	if target > numSegs {
		s.removeSegments((target - numSegs), allHosts)
	} else if target < numSegs {
		s.addSegments((numSegs - target), allHosts)
	}
}

func (s Seg) removeSegments(numSegs int, allHosts []string) {
	for _, host := range allHosts[:numSegs] {
		killSegment(host)
	}
}

func killSegment(address string) {
	_, err := http.Get(fmt.Sprintf("http://%s/shutdown", address))
	if err != nil {
		log.Fatal(err)
	}
}

func (s Seg) addSegments(numSegs int, allHosts []string) {
	var counter int

	for _, host := range allHosts {
		if ok := pingHost(host); !ok {
			s.SendSegment(host)
			counter += 1
		}

		if counter == numSegs {
			break
		}
	}
}

func (s Seg) SendSegment(address string) {

	url := fmt.Sprintf("http://%s%s/wormgate?sp=%s", address, s.wormgatePort, s.segmentPort)
	filename := "tmp.tar.gz"

	log.Printf("Spreading to %s", url)

	// ship the binary and the qml file that describes our screen output
	tarCmd := exec.Command("tar", "-zc", "-f", filename, "segment")
	tarCmd.Run()
	defer os.Remove(filename)

	file, err := os.Open(filename)
	if err != nil {
		log.Panic("Could not read input file", err)
	}

	resp, err := http.Post(url, "string", file)
	if err != nil {
		log.Panic("POST error ", err)
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Println("Received OK from server")
	} else {
		log.Println("Response: ", resp)
	}
}

func Run(wormPort, segPort, mode, spreadHost string) {
	hostName, _ := os.Hostname()

	s := &Seg{
		hostName:     hostName,
		segmentPort:  segPort,
		wormgatePort: wormPort,
	}

	switch mode {

	case "spread":
		s.SendSegment(spreadHost)
	case "run":
		s.StartSegmentServer()

	default:
		log.Fatalf("Unknown mode %q\n", os.Args[1])
	}
}
