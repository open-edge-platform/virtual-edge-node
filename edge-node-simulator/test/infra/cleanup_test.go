// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

var _ = Describe("Infrastructure Manager integration tests", Label(cleanupLabel), func() {
	var infraAPIClient *api.ClientWithResponses
	var cfg *flags_test.TestConfig
	var cancel context.CancelFunc
	var ctx context.Context
	var err error

	BeforeEach(func() {
		cfg = flags_test.GetConfig()
		Expect(cfg).NotTo(BeNil())

		ctx, cancel = context.WithCancel(context.Background())
		infraAPIClient, err = GetInfraAPIClient(ctx, cfg)
		Expect(err).To(BeNil())
	})
	AfterEach(func() {
		cancel()
	})

	Describe("Infrastructure Manager cleanup", Label(cleanupLabel), func() {
		It("should cleanup all hosts and locations in Infrastructure Manager", func(ctx SpecContext) {
			errCleanup := utils_test.HelperCleanupHosts(ctx, infraAPIClient)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupSchedules(ctx, infraAPIClient)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupLocations(ctx, infraAPIClient)
			Expect(errCleanup).To(BeNil())
		})
	})
})
