// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

func Test_GetAddresses(t *testing.T) {
	mac1, err := utils.GetRandomMACAddress()
	assert.NoError(t, err)
	assert.NotNil(t, mac1)
	mac2, err := utils.GetRandomMACAddress()
	assert.NoError(t, err)
	assert.NotNil(t, mac2)
	assert.NotEqual(t, mac1, mac2)
}
