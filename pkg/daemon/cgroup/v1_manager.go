package cgroup

import (
	"bytes"
	"context"
	"fmt"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/devices"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/configs"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

type v1Manager struct {
	cgroups.Manager
}

func newV1Manager(cg *configs.Cgroup, paths map[string]string) (Manager, error) {
	cgroupManager, err := fs.NewManager(cg, paths)
	if err != nil {
		return nil, err
	}
	return &v1Manager{
		Manager: cgroupManager,
	}, nil
}

func (m *v1Manager) Set(ctx context.Context, vm *virtv1alpha1.VirtualMachine, r *configs.Resources) error {
	devicesPath, ok := m.GetPaths()["devices"]
	if !ok {
		return fmt.Errorf("devices subsystem's path is not defined for this manager")
	}

	deviceRulesStr, err := cgroups.ReadFile(devicesPath, "devices.list")
	if err != nil {
		return fmt.Errorf("read deivces.list: %v", err)
	}

	emulator, err := devices.EmulatorFromList(bytes.NewBufferString(deviceRulesStr))
	if err != nil {
		return fmt.Errorf("create emulator: %v", err)
	}

	deviceRules, err := emulator.Rules()
	if err != nil {
		return fmt.Errorf("get device rules from emulator: %v", err)
	}

	resource := *r
	resource.Devices = append(resource.Devices, deviceRules...)
	return m.Manager.Set(&resource)
}
