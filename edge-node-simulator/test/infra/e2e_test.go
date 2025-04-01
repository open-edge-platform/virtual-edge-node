// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInfrastructureManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Infrastructure Manager E2E Integration Suite")
}
