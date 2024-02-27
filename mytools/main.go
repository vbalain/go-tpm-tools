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
	defer tpm.Close()

	// check AK (EK signing) cert
	gceAk, err := client.GceAttestationKeyECC(tpm)
	if err != nil {
		fmt.Printf("vaibhav 2 %v", err)
		return
	}
	if gceAk.Cert() == nil {
		fmt.Printf("vaibhav 3 failed to find AKCert on this VM: try creating a new VM or contacting support")
		return
	}
	gceAk.Close()
	fmt.Printf("vaibhav 4")
}
