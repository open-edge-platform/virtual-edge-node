// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

var _ = Describe("Infrastructure Manager integration tests", Label(cleanupLabel), func() {
	var apiClient *http.Client
	var cfg *flags_test.TestConfig

	BeforeEach(func() {
		cfg = flags_test.GetConfig()
		Expect(cfg).NotTo(BeNil())

		certCA, err := utils_test.LoadFile(cfg.CAPath)
		Expect(err).To(BeNil())

		apiClient, err = utils_test.GetHTTPClientWithCA(certCA)
		Expect(err).To(BeNil())
	})

	Describe("Infrastructure Manager cleanup", Label(cleanupLabel), func() {
		It("should cleanup all hosts and locations in Infrastructure Manager", func(ctx SpecContext) {
			err := utils_test.HelperCleanupHostsAPI(ctx, apiClient, cfg)
			Expect(err).To(BeNil())
		})
	})
})
