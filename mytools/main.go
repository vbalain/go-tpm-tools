package main

import (
	"fmt"

	"github.com/google/go-tpm-tools/client"
	"github.com/google/go-tpm/legacy/tpm2"
)

func main() {
	fmt.Printf("vaibhav 0")
	tpm, err := tpm2.OpenTPM("/dev/tpmrm0")
	if err != nil {
		fmt.Printf("vaibhav 1 %v", err)
		return
	}
	fmt.Printf("vaibhav 2 %v", tpm)
	defer tpm.Close()

	// check AK (EK signing) cert
	gceAk, err := client.GceAttestationKeyECC(tpm)
	if err != nil {
		fmt.Printf("vaibhav 3 %v", err)
		return
	}
	fmt.Printf("vaibhav 4 %v", gceAk)
	if gceAk.Cert() == nil {
		fmt.Printf("vaibhav 5 failed to find AKCert on this VM: try creating a new VM or contacting support")
		return
	}
	fmt.Printf("vaibhav 6 %v", gceAk.Cert())
	gceAk.Close()
	fmt.Printf("vaibhav 7")
}
