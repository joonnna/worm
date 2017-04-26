package util

import (
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	_ "log"
	"math/big"
	"net/http"
	"strings"
)

func ComputeHash(addr string) big.Int {
	h := sha1.New()
	io.WriteString(h, addr)
	hashBytes := h.Sum(nil)

	ret := big.NewInt(0)
	ret.SetBytes(hashBytes)
	return *ret

}

func CmpHash(h1, h2 big.Int) int {
	return h1.Cmp(&h2)
}

func FetchReachableHosts(port, hostName string) []string {
	url := fmt.Sprintf("http://localhost%s/reachablehosts", port)
	resp, err := http.Get(url)
	if err != nil {
		return []string{}
	}

	var bytes []byte
	bytes, err = ioutil.ReadAll(resp.Body)
	body := string(bytes)
	resp.Body.Close()

	trimmed := strings.TrimSpace(body)
	nodes := strings.Split(trimmed, "\n")

	for idx, host := range nodes {

		if host == hostName {
			nodes = append(nodes[:idx], nodes[idx+1:]...)
			break
		}
	}

	return nodes
}

func SliceDiff(s1, s2 []string) []string {
	var retSlice []string
	m := make(map[string]bool)

	for _, s := range s1 {
		m[s] = true
	}

	for _, s := range s2 {
		if _, ok := m[s]; !ok {
			retSlice = append(retSlice, s)
		}
	}

	return retSlice
}
