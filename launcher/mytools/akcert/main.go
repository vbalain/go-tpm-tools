package main

import (
	"fmt"

	"github.com/google/go-tpm-tools/client"
	"github.com/google/go-tpm/legacy/tpm2"
)

func main() {
	tpm, err := tpm2.OpenTPM("/dev/tpmrm0")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer tpm.Close()

	// check AK (EK signing) cert
	gceAk, err := client.GceAttestationKeyECC(tpm)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(gceAk)
	if gceAk.Cert() == nil {
		fmt.Println("failed to find AKCert on this VM: try creating a new VM or contacting support")
		return
	}
	fmt.Println(gceAk.Cert())
	gceAk.Close()
}
