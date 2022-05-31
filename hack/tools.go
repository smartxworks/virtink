//go:build tools
// +build tools

/*
 * Copyright (C) 2022 SmartX, Inc. <info@smartx.com>
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package tools

import (
	_ "github.com/golang/mock/mockgen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
