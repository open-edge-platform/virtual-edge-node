// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
)

const (
	scaleLabel     = "scale"
	scaleUpLabel   = "scaleup"
	scaleDownLabel = "scaledown"
	scaleCheck     = "scalecheck"
)

var (
	waitUntilHostsRunning = time.Second * 5
	waitUntilHostsDeleted = time.Second * 10
	filterRunning         = fmt.Sprintf(`%s = %q`, "host_status", "Running")
)

var _ = Describe("Infrastructure Manager integration scale tests", Label(scaleLabel), func() {
	var ensimClient ensim.Client
	var httpClient *http.Client
	var cancel context.CancelFunc
	var ctx context.Context
	var cfg *flags_test.TestConfig

	BeforeEach(func() {
		cfg = flags_test.GetConfig()
		Expect(cfg).NotTo(BeNil())

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

		ensimClient, err = ensim.NewClient(ctx, cfg.ENSimAddress)
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

	Describe("Scale Up", Label(scaleUpLabel), func() {
		It("should do NIO by create/verify edge nodes with Infrastructure Manager simulator", func(ctx SpecContext) {
			By("creating edge nodes in Infrastructure Manager simulator")
			enCredentals := &ensimapi.NodeCredentials{
				Project:         cfg.Project,
				OnboardUsername: cfg.EdgeOnboardUser,
				OnboardPassword: cfg.EdgeOnboardPass,
				ApiUsername:     cfg.EdgeAPIUser,
				ApiPassword:     cfg.EdgeAPIPass,
			}
			errCreate := ensimClient.CreateNodes(ctx, uint32(cfg.AmountEdgeNodes), uint32(cfg.BatchEdgeNodes), enCredentals, true)
			Expect(errCreate).To(BeNil())

			By("waiting for edge nodes to be created in Infrastructure Manager simulator")
			time.Sleep(waitUntilHostsRunning)

			By("checking edge nodes state/status ok from Infrastructure Manager simulator")
			listNodes, err := ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(BeNumerically(">=", cfg.AmountEdgeNodes))

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

			By("checking no hosts exist in Infrastructure Manager REST API")
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(BeNumerically(">=", cfg.AmountEdgeNodes))
		})
	})

	Describe("Scale Check", Label(scaleCheck), func() {
		It("should do a check of running edge nodes with Infrastructure Manager simulator", func(ctx SpecContext) {
			By("checking edge nodes state/status ok from Infrastructure Manager simulator")
			listNodes, err := ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(BeNumerically(">=", cfg.AmountEdgeNodes))

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

			By("checking no hosts exist in Infrastructure Manager REST API")
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, &filterRunning)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(BeNumerically(">=", cfg.AmountEdgeNodes))
		})
	})

	Describe("Scale Down", Label(scaleDownLabel), func() {
		It("should do delete of edge nodes with Infrastructure Manager simulator", func(ctx SpecContext) {
			By("checking no hosts exist in Infrastructure Manager REST API")
			totalHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, nil)
			Expect(err).To(BeNil())
			Expect(totalHosts).To(BeNumerically(">=", cfg.AmountEdgeNodes))

			By("checking edge nodes state/status ok from Infrastructure Manager simulator")
			listNodes, err := ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(BeNumerically(">=", cfg.AmountEdgeNodes))

			By("deleting all edge nodes from Infrastructure Manager simulator")
			err = ensimClient.DeleteNodes(ctx, uint32(cfg.AmountEdgeNodes))
			Expect(err).To(BeNil())

			By("waiting for edge nodes to be deleted from Infrastructure Manager simulator")
			time.Sleep(waitUntilHostsDeleted)

			lessNodes := totalHosts - cfg.AmountEdgeNodes
			By("checking no edge nodes exist in Infrastructure Manager simulator")
			listNodes, err = ensimClient.List(ctx)
			Expect(err).To(BeNil())
			Expect(len(listNodes)).To(BeNumerically("==", lessNodes))

			By("checking no hosts exist in Infrastructure Manager REST API")
			remainingHosts, err := utils_test.ListHostsTotalAPI(ctx, httpClient, cfg, nil)
			Expect(err).To(BeNil())
			Expect(remainingHosts).To(BeNumerically("==", lessNodes))
		})
	})
})
