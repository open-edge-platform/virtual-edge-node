// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

var _ = Describe("Infrastructure Manager integration tests", Label(cleanupLabel), func() {
	var cfg *flags_test.TestConfig
	var httpClient *http.Client
	var cancel context.CancelFunc
	var ctx context.Context

	BeforeEach(func() {
		cfg = flags_test.GetConfig()
		Expect(cfg).NotTo(BeNil())

		certCA, err := utils_test.LoadFile(cfg.CAPath)
		Expect(err).To(BeNil())

		httpClient, err = utils_test.GetClientWithCA(certCA)
		Expect(err).To(BeNil())

		ctx, cancel = context.WithCancel(context.Background())
		Expect(ctx).NotTo(BeNil())
		Expect(cancel).NotTo(BeNil())

		err = utils_test.HelperJWTTokenRoutine(ctx, certCA, cfg.OrchFQDN, cfg.EdgeAPIUser, cfg.EdgeAPIPass)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		cancel()
	})

	Describe("Infrastructure Manager cleanup", Label(cleanupLabel), func() {
		It("should cleanup all hosts and locations in Infrastructure Manager", func(ctx SpecContext) {
			errCleanup := utils_test.HelperCleanupHostsAPI(ctx, httpClient, cfg)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupSchedulesAPI(ctx, httpClient, cfg)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupLocationsAPI(ctx, httpClient, cfg)
			Expect(errCleanup).To(BeNil())
		})
	})
})
