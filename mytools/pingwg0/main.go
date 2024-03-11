package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/go-tpm-tools/mytools/showwg0"
)

var (
	ping_ip = flag.String("ip", "10.128.0.8", "instance-svm-1") // instance-svm-2 10.128.0.7
)

func main() {
	flag.Parse()

	ping(*ping_ip)

	showwg0.ShowConfig()
}

func ping(ip string) {
	out, _ := exec.Command("ping", ip, "-c 5", "-i 3", "-w 10").Output()
	fmt.Println("ping output: ", out)
	if strings.Contains(string(out), "Destination Host Unreachable") {
		fmt.Println("TANGO DOWN")
	} else {
		fmt.Println("IT'S ALIVEEE")
	}
}
