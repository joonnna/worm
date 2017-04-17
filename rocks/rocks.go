package rocks

import (
	"log"
	"os/exec"
	"strings"
)

func ListNodes() []string {
	cmdline := []string{"bash", "-c",
		"rocks list host compute | cut -d : -f1 | sed 1d"}
	log.Printf("Getting list of nodes: %q", cmdline)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	out, err := cmd.Output()
	if err != nil {
		log.Panic("Error getting available nodes", err)
	}

	trimmed := strings.TrimSpace(string(out))
	nodes := strings.Split(trimmed, "\n")
	return nodes
}
