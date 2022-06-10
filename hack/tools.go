//go:build tools
// +build tools

package tools

import (
	_ "github.com/golang/mock/mockgen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
