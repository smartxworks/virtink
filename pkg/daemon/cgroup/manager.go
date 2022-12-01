package cgroup

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/configs"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
	"github.com/smartxworks/virtink/pkg/daemon/pid"
)

type Manager interface {
	Set(ctx context.Context, vm *virtv1alpha1.VirtualMachine, r *configs.Resources) error
}

func NewManager(ctx context.Context, vm *virtv1alpha1.VirtualMachine) (Manager, error) {
	cg := &configs.Cgroup{
		Resources: &configs.Resources{},
	}

	socketPath := filepath.Join("/var/lib/kubelet/pods", string(vm.Status.VMPodUID), "volumes/kubernetes.io~empty-dir/virtink/", "ch.sock")
	if _, err := os.Stat(socketPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	vmPID, err := pid.GetPIDBySocket(socketPath)
	if err != nil {
		return nil, err
	}
	cgroupPaths, err := cgroups.ParseCgroupFile(fmt.Sprintf("/proc/%v/cgroup", vmPID))
	if err != nil {
		return nil, err
	}

	if cgroups.IsCgroup2UnifiedMode() {
		cgroupPath := cgroupPaths[""]
		err := filepath.Walk("/proc/1/root/sys/fs/cgroup/kubepods.slice", func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() {
				return nil
			}
			if filepath.Base(path) == filepath.Base(cgroupPath) {
				cgroupPath = path
				return io.EOF
			}
			return nil
		})
		if err != nil && err != io.EOF {
			return nil, err
		}
		return newV2Manager(cg, cgroupPath, vmPID)
	} else {
		for subsystem, path := range cgroupPaths {
			if subsystem != "" {
				cgroupPaths[subsystem] = filepath.Join("/proc/1/root/sys/fs/cgroup", subsystem, path)
			}
		}
		return newV1Manager(cg, cgroupPaths)
	}
}
