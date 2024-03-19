package main

import (
	"fmt"

	"github.com/google/go-tpm-tools/client"
	"github.com/google/go-tpm/legacy/tpm2"
)

func main() {
	fmt.Println("vaibhav 0 ")
	tpm, err := tpm2.OpenTPM("/dev/tpmrm0")
	if err != nil {
		fmt.Println("vaibhav 1 ", err)
		return
	}
	fmt.Println("vaibhav 2 ", tpm)
	defer tpm.Close()

	// check AK (EK signing) cert
	gceAk, err := client.GceAttestationKeyECC(tpm)
	if err != nil {
		fmt.Println("vaibhav 3 ", err)
		return
	}
	fmt.Println("vaibhav 4 ", gceAk)
	if gceAk.Cert() == nil {
		fmt.Println("vaibhav 5 failed to find AKCert on this VM: try creating a new VM or contacting support")
		return
	}
	fmt.Println("vaibhav 6 ", gceAk.Cert())
	gceAk.Close()
	fmt.Println("vaibhav 7")
}
