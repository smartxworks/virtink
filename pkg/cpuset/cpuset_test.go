package cpuset_test

import (
	"testing"

	assert "github.com/stretchr/testify/require"

	"github.com/smartxworks/virtink/pkg/cpuset"
)

func TestParse(t *testing.T) {
	positiveTestCases := []struct {
		cpusetString string
		expected     cpuset.CPUSet
	}{
		{"", cpuset.NewCPUSet()},
		{"5", cpuset.NewCPUSet(5)},
		{"1,2,3,4,5", cpuset.NewCPUSet(1, 2, 3, 4, 5)},
		{"1-5", cpuset.NewCPUSet(1, 2, 3, 4, 5)},
		{"1-2,3-5", cpuset.NewCPUSet(1, 2, 3, 4, 5)},
		{"5,4,3,2,1", cpuset.NewCPUSet(1, 2, 3, 4, 5)},
		{"3-6,1-5", cpuset.NewCPUSet(1, 2, 3, 4, 5, 6)},
		{"3-3,5-5", cpuset.NewCPUSet(3, 5)},
	}

	for _, c := range positiveTestCases {
		result, err := cpuset.Parse(c.cpusetString)
		assert.NoError(t, err)
		assert.Equal(t, c.expected, result)
	}

	negativeTestCases := []string{
		"nonnumeric", "non-numeric", "no,numbers", "0-a", "a-0", "0,a", "a,0", "1-2,a,3-5",
		"0,", "0,,", ",3", ",,3", "0,,3",
		"-1", "1-", "1,2-,3", "1,-2,3", "-1--2", "--1", "1--",
		"3-0", "0--3"}
	for _, c := range negativeTestCases {
		_, err := cpuset.Parse(c)
		assert.Error(t, err)
	}
}
