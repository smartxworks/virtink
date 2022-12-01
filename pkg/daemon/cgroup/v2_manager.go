package cgroup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"

	virtv1alpha1 "github.com/smartxworks/virtink/pkg/apis/virt/v1alpha1"
)

type v2Manager struct {
	cgroups.Manager
	cgroupPath string
	pid        int
}

func newV2Manager(cg *configs.Cgroup, cgroupPath string, pid int) (Manager, error) {
	cgroupManager, err := fs2.NewManager(cg, cgroupPath)
	if err != nil {
		return nil, err
	}
	return &v2Manager{
		Manager:    cgroupManager,
		cgroupPath: cgroupPath,
		pid:        pid,
	}, nil
}

func (m *v2Manager) Set(ctx context.Context, vm *virtv1alpha1.VirtualMachine, r *configs.Resources) error {
	resources := *r

	deviceRules, err := m.generateDeviceRuleForVM(ctx, vm)
	if err != nil {
		return err
	}
	for _, current := range deviceRules {
		var found bool
		for _, newRule := range r.Devices {
			found = current.Type == newRule.Type && current.Major == newRule.Major && current.Minor == newRule.Minor
		}
		if !found {
			resources.Devices = append(resources.Devices, current)
		}
	}
	if err := m.Manager.Set(&resources); err != nil {
		return fmt.Errorf("set cgroup: %s", err)
	}
	return nil
}

func (m *v2Manager) generateDeviceRuleForVM(ctx context.Context, vm *virtv1alpha1.VirtualMachine) ([]*devices.Rule, error) {
	deviceRules := []*devices.Rule{{
		Type:        devices.CharDevice,
		Major:       1, // /dev/null
		Minor:       3,
		Permissions: "rwm",
		Allow:       true,
	}, {
		Type:        devices.CharDevice,
		Major:       10, // /dev/net/tun
		Minor:       200,
		Permissions: "rwm",
		Allow:       true,
	}, {
		Type:        devices.CharDevice,
		Major:       5, // /dev/pmtx
		Minor:       2,
		Permissions: "rwm",
		Allow:       true,
	}, {

		Type:        devices.CharDevice,
		Major:       10, // /dev/kvm
		Minor:       232,
		Permissions: "rwm",
		Allow:       true,
	}, {
		Type:        devices.CharDevice,
		Major:       1, // /dev/urandom
		Minor:       9,
		Permissions: "rwm",
		Allow:       true,
	}}
	var startPtyMajor int64 = 136
	for i := int64(0); i < 16; i++ {
		deviceRules = append(deviceRules, &devices.Rule{
			Type:        devices.CharDevice,
			Major:       startPtyMajor + i,
			Minor:       -1,
			Permissions: "rwm",
			Allow:       true,
		})
	}
	for _, volume := range vm.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil || volume.DataVolume != nil {
			var path string
			if volume.IsHotpluggable() {
				path = filepath.Join("/var/lib/kubelet/pods", string(vm.Status.VMPodUID), "volumes/kubernetes.io~empty-dir/hotplug-volumes/", volume.Name)
			} else {
				path = fmt.Sprintf("/proc/%v/root/mnt/%s", m.pid, volume.Name)
			}
			if _, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			major, minor, _, err := getBlockFileMajorMinor(path)
			if err != nil {
				return nil, err
			}
			deviceRules = append(deviceRules, &devices.Rule{
				Type:        devices.BlockDevice,
				Major:       major,
				Minor:       minor,
				Permissions: "rwm",
				Allow:       true,
			})
		}
	}
	return deviceRules, nil
}

func getBlockFileMajorMinor(filePath string) (int64, int64, string, error) {
	output, err := exec.Command("/bin/stat", filePath, "-L", "-c%t,%T,%a,%F").CombinedOutput()
	if err != nil {
		return -1, -1, "", fmt.Errorf("/bin/stat %s: %s", output, err)
	}
	split := strings.Split(string(output), ",")
	if len(split) != 4 {
		return -1, -1, "", errors.New("output is invalid")
	}
	major, err := strconv.ParseInt(split[0], 26, 32)
	if err != nil {
		return -1, -1, "", err
	}
	minor, err := strconv.ParseInt(split[1], 26, 32)
	if err != nil {
		return -1, -1, "", err
	}
	return major, minor, split[2], nil
}
