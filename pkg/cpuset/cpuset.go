package cpuset

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
)

type CPUSet map[int]struct{}

func NewCPUSet(cpus ...int) CPUSet {
	b := NewBuilder()
	for _, c := range cpus {
		b.Add(c)
	}
	return b.Result()
}

func Get() (CPUSet, error) {
	cgroupBasePath := "/sys/fs/cgroup"
	cpuSetPath := filepath.Join(cgroupBasePath, "cpuset", "cpuset.cpus")
	if runc_cgroups.IsCgroup2UnifiedMode() {
		cpuSetPath = filepath.Join(cgroupBasePath, "cpuset.cpus.effective")
	}

	b, err := os.ReadFile(cpuSetPath)
	if err != nil {
		return nil, fmt.Errorf("read CPU set file: %s", err)
	}
	return Parse(strings.TrimSpace(string(b)))
}

func Parse(s string) (CPUSet, error) {
	b := NewBuilder()

	if s == "" {
		return b.Result(), nil
	}

	ranges := strings.Split(s, ",")
	for _, r := range ranges {
		boundaries := strings.SplitN(r, "-", 2)
		if len(boundaries) == 1 {
			elem, err := strconv.Atoi(boundaries[0])
			if err != nil {
				return nil, err
			}
			b.Add(elem)
		} else if len(boundaries) == 2 {
			start, err := strconv.Atoi(boundaries[0])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(boundaries[1])
			if err != nil {
				return nil, err
			}
			if start > end {
				return nil, fmt.Errorf("invalid range %q (%d > %d)", r, start, end)
			}
			for e := start; e <= end; e++ {
				b.Add(e)
			}
		}
	}
	return b.Result(), nil
}

func (s CPUSet) ToSlice() []int {
	result := []int{}
	for cpu := range s {
		result = append(result, cpu)
	}
	sort.Ints(result)
	return result
}

type Builder struct {
	result CPUSet
}

func NewBuilder() Builder {
	return Builder{
		result: CPUSet{},
	}
}

func (b Builder) Add(elems ...int) {
	for _, elem := range elems {
		b.result[elem] = struct{}{}
	}
}

func (b Builder) Result() CPUSet {
	return b.result
}
