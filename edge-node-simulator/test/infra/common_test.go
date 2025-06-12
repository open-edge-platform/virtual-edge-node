// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
)

var zlog = logging.GetLogger("en-test")

var (
	e2eLabel     = "infra-e2e"
	day2Label    = "infra-tests-day2"
	day1Label    = "infra-tests-day1"
	day0Label    = "infra-tests-day0"
	cleanupLabel = "cleanup"
)

var (
	waitUntilHostsRunning   = time.Second * 5
	waitUntilHostsDeleted   = time.Second * 2
	waitHostsRunning        = time.Minute * 5
	waitHostsConnectionLost = time.Minute * 5
	waitHostsMaintenance    = time.Minute * 1

	TimeNow       = int(time.Now().UTC().Unix())
	SafeTimeDelay = 600
)

var (
	filterRunning             = fmt.Sprintf(`%s=%q`, "host_status", "Running")
	filterNoConnection        = fmt.Sprintf(`%s=%q`, "host_status", "No Connection")
	filterInstanceStatusError = fmt.Sprintf(`%s=%q`, "instance_status", "Error")
)

func GenerateUUIDs(cfg *flags_test.TestConfig) []string {
	// Create nodes in Infrastructure Manager SIM
	enUUIDs := []string{}
	for i := 0; i < cfg.AmountEdgeNodes; i++ {
		hostUUID := uuid.New()
		enUUID := hostUUID.String()
		enUUIDs = append(enUUIDs, enUUID)
	}
	return enUUIDs
}

func GetENSimClient(ctx context.Context, cfg *flags_test.TestConfig) (ensim.Client, error) {
	simClient, err := ensim.NewClient(ctx, cfg.ENSimAddress)
	return simClient, err
}

func ENSIMCheckNodes(ctx context.Context, simClient ensim.Client, amount int) error {
	listNodes, err := simClient.List(ctx)
	if amount != len(listNodes) {
		return err
	}
	return nil
}

func ENSIMCreateNodes(ctx context.Context,
	cfg *flags_test.TestConfig,
	simClient ensim.Client,
	enUUIDs []string,
) error {
	enCredentals := &ensimapi.NodeCredentials{
		Project:         cfg.Project,
		OnboardUsername: cfg.EdgeOnboardUser,
		OnboardPassword: cfg.EdgeOnboardPass,
		ApiUsername:     cfg.EdgeAPIUser,
		ApiPassword:     cfg.EdgeAPIPass,
	}
	for _, enUUID := range enUUIDs {
		zlog.Info().Msgf("Creating node %v", enUUID)
		err := simClient.Create(ctx, enUUID, enCredentals, true)
		if err != nil {
			return err
		}
	}
	return nil
}
