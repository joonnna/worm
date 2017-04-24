package main

import (
	"flag"
	_ "fmt"
	"github.com/joonnna/worm/segment"
	"github.com/joonnna/worm/visualize"
	"github.com/joonnna/worm/wormgate"
	"log"
	"os"
)

func main() {

	var hostname, segPort, wormPort, host, mode, progType string
	var targetSegments int

	hostname, _ = os.Hostname()
	log.SetPrefix(hostname + " segment: ")

	args := flag.NewFlagSet("args", flag.ExitOnError)
	addFlags(args, &wormPort, &segPort, &mode, &host, &progType, &targetSegments)

	args.Parse(os.Args[1:])

	switch progType {

	case "visualizer":
		visualize.Run(wormPort, segPort)
	case "wormgate":
		wormgate.Run(wormPort)
	case "segment":
		segment.Run(wormPort, segPort, mode, host, targetSegments)

	default:
		log.Fatalf("Unknown mode %q\n", os.Args[1])
	}

}

func addFlags(flagset *flag.FlagSet, wormPort, segPort, mode, host, progType *string, target *int) {
	flagset.StringVar(wormPort, "wp", ":8181", "wormgate port (prefix with colon)")
	flagset.StringVar(segPort, "sp", ":8182", "segment port (prefix with colon)")
	flagset.StringVar(progType, "prog", "visualizer", "program to run")
	flagset.StringVar(mode, "mode", "run", "segment mode")
	flagset.StringVar(host, "host", "compute-1-0", "host to spread to")
	flagset.IntVar(target, "target", 5, "segment target number")
}
