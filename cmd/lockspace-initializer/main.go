package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/namsral/flag"

	"github.com/smartxworks/virtink/pkg/sanlock"
)

func main() {
	var lockspaceName string
	var ioTimeoutSeconds int
	flag.StringVar(&lockspaceName, "lockspace-name", "", "")
	flag.IntVar(&ioTimeoutSeconds, "io-timeout-seconds", 10, "")
	flag.Parse()

	leaseFilePath := filepath.Join("/var/lib/sanlock", lockspaceName, "leases")
	if _, err := os.Stat(leaseFilePath); err != nil {
		if os.IsNotExist(err) {
			if _, err := exec.Command("touch", leaseFilePath).CombinedOutput(); err != nil {
				log.Fatalf("create lease file: %s", err)
			}
		} else {
			log.Fatalf("check lease file status: %s", err)
		}
	}

	if err := sanlock.WriteLockspaceWithIOTimeout(lockspaceName, leaseFilePath, uint32(ioTimeoutSeconds)); err != nil {
		log.Fatalf("create Sanlock Lockspace: %s", err)
	}

	if err := sanlock.FormatRIndex(lockspaceName, leaseFilePath); err != nil {
		log.Fatalf("format Sanlock RIndex: %s", err)
	}
}
