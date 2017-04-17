package util

import (
    "math/big"
    "crypto/sha1"
    "fmt"
    "io"
    "io/ioutil"
    "strings"
    "net/http"
    "log"
)




func ComputeHash(addr string) *big.Int {
    h := sha1.New()
    io.WriteString(h, addr)
    hashBytes := h.Sum(nil)

    ret := big.NewInt(0)
    ret.SetBytes(hashBytes)
    return ret

}


func CmpHash(h1, h2 *big.Int) bool {
    cmp := h1.Cmp(h2)  
    if cmp == 1 {
        return true
    } else if cmp == -1 {
        return false
    } else {
        log.Fatal("Equal hash values?!")
        return false
    }
}


func FetchReachableHosts(port string) []string {
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
	return nodes
}
