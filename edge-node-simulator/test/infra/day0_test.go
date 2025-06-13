// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

var _ = Describe("Infrastructure Manager integration tests", Label(e2eLabel), func() {
	var ensimClient ensim.Client
	var httpClient *http.Client
	var cancel context.CancelFunc
	var ctx context.Context
	var cfg *flags_test.TestConfig
	var enUUIDs1 []string
	var enUUIDs2 []string

	BeforeEach(func() {
		cfg = flags_test.GetConfig()
		Expect(cfg).NotTo(BeNil())
	})

	JustBeforeEach(func() {
		var err error
		certCA, err := utils_test.LoadFile(cfg.CAPath)
		Expect(err).To(BeNil())

		httpClient, err = utils_test.GetClientWithCA(certCA)
		Expect(err).To(BeNil())

		ctx, cancel = context.WithCancel(context.Background())
		Expect(ctx).NotTo(BeNil())
		Expect(cancel).NotTo(BeNil())

		err = utils_test.HelperJWTTokenRoutine(ctx, certCA, cfg.ClusterFQDN, cfg.EdgeAPIUser, cfg.EdgeAPIPass)
		Expect(err).To(BeNil())

		ctx, cancel = context.WithCancel(context.Background())
		Expect(ctx).NotTo(BeNil())
		Expect(cancel).NotTo(BeNil())

		err = utils_test.HelperJWTTokenRoutine(ctx, certCA, cfg.ClusterFQDN, cfg.EdgeAPIUser, cfg.EdgeAPIPass)
		Expect(err).To(BeNil())

		ensimClient, err = GetENSimClient(ctx, cfg)
		Expect(err).To(BeNil())

		if cfg.Cleanup {
			errCleanup := utils_test.HelperCleanupHostsAPI(ctx, httpClient, cfg)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupSchedulesAPI(ctx, httpClient, cfg)
			Expect(errCleanup).To(BeNil())
			errCleanup = utils_test.HelperCleanupLocationsAPI(ctx, httpClient, cfg)
			Expect(errCleanup).To(BeNil())
		}
	})

	AfterEach(func() {
		cancel()
	})

	Describe("day0 - Non Interactive Onboarding (NIO)", Label(day0Label), func() {
		BeforeEach(func() {
			enUUIDs1 = GenerateUUIDs(cfg)
			Expect(enUUIDs1).NotTo(BeNil())
		})
		It("should do NIO by create/check/delete/verify edge nodes with Infrastructure Manager simulator", func(ctx SpecContext) {
			By("creating edge nodes in Infrastructure Manager simulator")
			enCredentals := &ensimapi.NodeCredentials{
				Project:         cfg.Project,
				OnboardUsername: cfg.EdgeOnboardUser,
				OnboardPassword: cfg.EdgeOnboardPass,
				ApiUsername:     cfg.EdgeAPIUser,
				ApiPassword:     cfg.EdgeAPIPass,
			}
			for _, enUUID := range enUUIDs1 {
				zlog.Info().Msgf("Creating node %v", enUUID)
				errCreate := ensimClient.Create(ctx, enUUID, enCredentals, true)
				Expect(errCreate).To(BeNil())
			}

			By("checking edge nodes state/status ok from Infrastructure Manager simulator")
			listNodes, err := ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(Equal(cfg.AmountEdgeNodes))

			for _, simNode := range listNodes {
				Expect(simNode.Credentials.Project).To(Equal(cfg.Project))
				Expect(simNode.Credentials.OnboardUsername).To(Equal(cfg.EdgeOnboardUser))
				Expect(simNode.Credentials.OnboardPassword).To(Equal(cfg.EdgeOnboardPass))

				for _, state := range simNode.AgentsStates {
					Expect(state.CurrentState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
					Expect(state.DesiredState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
				}
				for _, status := range simNode.Status {
					Expect(status.GetMode().String()).To(Equal(ensimapi.StatusMode_STATUS_MODE_OK.String()),
						"Status mode is not OK for %v", status.Source.String())
				}
			}
			time.Sleep(waitUntilHostsRunning)

			By("checking all hosts in running status from Infrastructure Manager REST API")
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))

			By("checking all hosts status from Infrastructure Manager REST API")
			for _, enUUID := range enUUIDs1 {
				utils_test.CheckHostStatusAPI(ctx, GinkgoTB(), httpClient, cfg, enUUID)
			}

			By("deleting all edge nodes from Infrastructure Manager simulator")
			err = ensimClient.DeleteNodes(ctx, 0)
			Expect(err).To(BeNil())

			By("checking no edge nodes exist in Infrastructure Manager simulator")
			listNodes, err = ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(Equal(0))

			By("checking no hosts exist in Infrastructure Manager REST API")
			totalHosts, err = utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(0))
		})
	})
	Describe("day0 - Non Interactive Onboarding (NIO) - onboard only", Label(day0CreateLabel), func() {
		BeforeEach(func() {
			enUUIDs2 = GenerateUUIDs(cfg)
			Expect(enUUIDs2).NotTo(BeNil())
		})
		It("should onboard edge nodes and verify they are up", func(ctx SpecContext) {
			By("creating edge nodes in Infrastructure Manager simulator")
			enCredentals := &ensimapi.NodeCredentials{
				Project:         cfg.Project,
				OnboardUsername: cfg.EdgeOnboardUser,
				OnboardPassword: cfg.EdgeOnboardPass,
				ApiUsername:     cfg.EdgeAPIUser,
				ApiPassword:     cfg.EdgeAPIPass,
			}
			for _, enUUID := range enUUIDs2 {
				zlog.Info().Msgf("Creating node %v", enUUID)
				errCreate := ensimClient.Create(ctx, enUUID, enCredentals, true)
				Expect(errCreate).To(BeNil())
			}

			By("checking edge nodes state/status ok from Infrastructure Manager simulator")
			listNodes, err := ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(Equal(cfg.AmountEdgeNodes))

			for _, simNode := range listNodes {
				Expect(simNode.Credentials.Project).To(Equal(cfg.Project))
				Expect(simNode.Credentials.OnboardUsername).To(Equal(cfg.EdgeOnboardUser))
				Expect(simNode.Credentials.OnboardPassword).To(Equal(cfg.EdgeOnboardPass))

				for _, state := range simNode.AgentsStates {
					Expect(state.CurrentState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
					Expect(state.DesiredState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
				}
				for _, status := range simNode.Status {
					Expect(status.GetMode().String()).To(Equal(ensimapi.StatusMode_STATUS_MODE_OK.String()),
						"Status mode is not OK for %v", status.Source.String())
				}
			}

			By("checking all hosts in running status from Infrastructure Manager REST API")
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))

			By("checking all hosts status from Infrastructure Manager REST API")
			for _, enUUID := range enUUIDs2 {
				utils_test.CheckHostStatusAPI(ctx, GinkgoTB(), httpClient, cfg, enUUID)
			}
		})
	})
	Describe("day0 - Non Interactive Onboarding (NIO) - delete only", Label(day0DeleteLabel), func() {
		BeforeEach(func() {
			cfg.Cleanup = false
			Expect(enUUIDs2).NotTo(BeNil())
		})
		It("should verify existing edge nodes are up and remove them", func(ctx SpecContext) {
			By("checking edge nodes state/status ok from Infrastructure Manager simulator")
			listNodes, err := ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(Equal(cfg.AmountEdgeNodes))
			time.Sleep(waitUntilStatusAvailable)

			for _, simNode := range listNodes {
				Expect(simNode.Credentials.Project).To(Equal(cfg.Project))
				Expect(simNode.Credentials.OnboardUsername).To(Equal(cfg.EdgeOnboardUser))
				Expect(simNode.Credentials.OnboardPassword).To(Equal(cfg.EdgeOnboardPass))

				for _, state := range simNode.AgentsStates {
					Expect(state.CurrentState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
					Expect(state.DesiredState).To(Equal(ensimapi.AgentState_AGENT_STATE_ON))
				}
				for _, status := range simNode.Status {
					Expect(status.GetMode().String()).To(Equal(ensimapi.StatusMode_STATUS_MODE_OK.String()),
						"Status mode is not OK for %v", status.Source.String())
				}
			}

			By("checking all hosts in running status from Infrastructure Manager REST API")
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(cfg.AmountEdgeNodes))

			By("checking all hosts status from Infrastructure Manager REST API")
			for _, enUUID := range enUUIDs2 {
				utils_test.CheckHostStatusAPI(ctx, GinkgoTB(), httpClient, cfg, enUUID)
			}

			By("deleting all edge nodes from Infrastructure Manager simulator")
			err = ensimClient.DeleteNodes(ctx, 0)
			Expect(err).To(BeNil())

			// Wait for the deletion to propagate
			time.Sleep(waitUntilHostsDeleted)

			By("checking no edge nodes exist in Infrastructure Manager simulator")
			listNodes, err = ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(Equal(0))

			By("checking no hosts exist in Infrastructure Manager REST API")
			totalHosts, err = utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(Equal(0))
		})
	})
})
