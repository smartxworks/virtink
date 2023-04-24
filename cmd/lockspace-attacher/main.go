package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/namsral/flag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/smartxworks/virtink/pkg/sanlock"
)

func main() {
	var lockspaceName string
	var nodeName string
	flag.StringVar(&lockspaceName, "lockspace-name", "", "")
	flag.StringVar(&nodeName, "node-name", "", "")
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		log.Fatalf("failed to build kubeconfig: %s", err)
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create Kubernetes client: %s", err)
	}

	hostID, err := getHostID(kubeClient, nodeName)
	if err != nil {
		log.Fatalf("failed to get host ID: %s", err)

	}

	leaseFilePath := filepath.Join("/var/lib/sanlock", lockspaceName, "leases")
	if err := sanlock.AcquireDeltaLease(lockspaceName, leaseFilePath, hostID); err != nil {
		log.Fatalf("failed to acquire delta lease: %s", err)
	}
	log.Println("succeeded to acquire delta lease")

	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1)
	}()

	<-stop
	if err := sanlock.ReleaseDeltaLease(lockspaceName, leaseFilePath, hostID); err != nil {
		if err != sanlock.ENOENT {
			log.Fatalf("failed to release delta lease: %s", err)
		}
	}
	if err := umountLeaseVolume(lockspaceName); err != nil {
		log.Fatalf("failed to umont lease volume: %s", err)
	}
}

//+kubebuilder:rbac:groups="",resources=nodes,verbs=get

func getHostID(client *kubernetes.Clientset, nodeName string) (uint64, error) {
	node, err := client.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("get node: %s", err)
	}

	if id, exist := node.Annotations["virtink.smartx.com/sanlock-host-id"]; exist {
		if hostID, err := strconv.ParseUint(id, 10, 64); err != nil {
			return 0, fmt.Errorf("parse uint: %s", err)
		} else {
			return hostID, nil
		}
	}
	return 0, fmt.Errorf("sanlock host %s ID not found", nodeName)
}

func umountLeaseVolume(lockspace string) error {
	leaseFileDir := filepath.Join("/var/lib/sanlock", lockspace)
	output, err := exec.Command("sh", "-c", fmt.Sprintf("mount | grep '%s'", leaseFileDir)).CombinedOutput()
	if err != nil && string(output) == "" {
		return nil
	}

	if _, err := exec.Command("umount", leaseFileDir).CombinedOutput(); err != nil {
		return fmt.Errorf("unmount volume: %s", err)
	}
	return nil
}
