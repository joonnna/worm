package main

import (
	"flag"
	"github.com/joonnna/worm/segment"
	"log"
	"os"
)

func main() {

	var hostname, segPort, wormPort string

	hostname, _ = os.Hostname()
	log.SetPrefix(hostname + " segment: ")

	var spreadMode = flag.NewFlagSet("spread", flag.ExitOnError)
	addCommonFlags(spreadMode, &wormPort, &segPort)
	var spreadHost = spreadMode.String("host", "localhost", "host to spread to")

	var runMode = flag.NewFlagSet("run", flag.ExitOnError)
	addCommonFlags(runMode, &wormPort, &segPort)

	if len(os.Args) == 1 {
		log.Fatalf("No mode specified\n")
	}

	s := &segment.Seg{
		HostName:     hostname,
		SegmentPort:  segPort,
		WormgatePort: wormPort,
	}

	switch os.Args[1] {
	case "spread":
		spreadMode.Parse(os.Args[2:])
		s.SendSegment(*spreadHost)
	case "run":
		runMode.Parse(os.Args[2:])
		s.StartSegmentServer()

	default:
		log.Fatalf("Unknown mode %q\n", os.Args[1])
	}
}

func addCommonFlags(flagset *flag.FlagSet, wormPort, segPort *string) {
	flagset.StringVar(wormPort, "wp", ":8181", "wormgate port (prefix with colon)")
	flagset.StringVar(segPort, "sp", ":8182", "segment port (prefix with colon)")
}
