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

	hostname, _ = os.Hostname()
	log.SetPrefix(hostname + " segment: ")

	flags := flag.NewFlagSet("args", flag.ExitOnError)
	addFlags(flags, &wormPort, &segPort, &mode, &host, &progType)

	flags.Parse(os.Args)
	/*
		var spreadMode = flag.NewFlagSet("spread", flag.ExitOnError)
		addCommonFlags(spreadMode, &wormPort, &segPort)
		var spreadHost = spreadMode.String("host", "localhost", "host to spread to")

		var runMode = flag.NewFlagSet("run", flag.ExitOnError)
		addCommonFlags(runMode, &wormPort, &segPort)
	*/
	if len(os.Args) == 1 {
		log.Fatalf("No mode specified\n")
	}

	switch progType {

	case "visualizer":
		visualize.Run(wormPort, segPort)
	case "wormgate":
		wormgate.Run(wormPort)
	case "segment":
		segment.Run(wormPort, segPort, mode, host)

	default:
		log.Fatalf("Unknown mode %q\n", os.Args[1])
	}

}

func addFlags(flagset *flag.FlagSet, wormPort, segPort, mode, host, progType *string) {
	flagset.StringVar(wormPort, "wp", ":8181", "wormgate port (prefix with colon)")
	flagset.StringVar(segPort, "sp", ":8182", "segment port (prefix with colon)")
	flagset.StringVar(progType, "prog", "visualizer", "program to run")
	flagset.StringVar(mode, "mode", "run", "segment mode")
	flagset.StringVar(host, "host", "compute-1-0", "host to spread to")
}
