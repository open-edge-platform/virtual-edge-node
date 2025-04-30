// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
	flags_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/flags"
	utils_test "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/test/utils"
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
	waitUntilHostsRunning   = time.Second * 2
	waitHostsRunning        = time.Minute * 5
	waitHostsConnectionLost = time.Minute * 5
	waitHostsMaintenance    = time.Minute * 1

	TimeNow       = int(time.Now().UTC().Unix())
	SafeTimeDelay = 600
)

var (
	filterRunning             = fmt.Sprintf(`%s = %q`, "host_status", "Running")
	filterNoConnection        = fmt.Sprintf(`%s = %q`, "host_status", "No Connection")
	filterInstanceStatusError = fmt.Sprintf(`%s = %q`, "instance_status", "Error")
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

func GetInfraAPIClient(ctx context.Context, cfg *flags_test.TestConfig) (*api.ClientWithResponses, error) {
	// Set the environment variables - projectID
	os.Setenv(utils.ProjectIDEnvVar, cfg.Project)

	certCA, err := utils_test.LoadFile(cfg.CAPath)
	if err != nil {
		return nil, err
	}

	err = utils_test.HelperJWTTokenRoutine(ctx, certCA, cfg.OrchFQDN, cfg.EdgeAPIUser, cfg.EdgeAPIPass)
	if err != nil {
		return nil, err
	}

	httpClient, err := utils_test.GetClientWithCA(certCA)
	if err != nil {
		return nil, err
	}

	apiClient, err := api.NewClientWithResponses(cfg.InfraRESTAPIAddress, api.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	return apiClient, nil
}

func GetENSimClient(ctx context.Context, cfg *flags_test.TestConfig) (ensim.Client, error) {
	simClient, err := ensim.NewClient(ctx, cfg.ENSimAddress)
	return simClient, err
}

func InfraAPICheckHosts(ctx context.Context,
	apiClient *api.ClientWithResponses,
	filter *string,
	amount int,
) error {
	resList, err := apiClient.GetComputeHostsWithResponse(
		ctx,
		&api.GetComputeHostsParams{
			Filter: filter,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)

	if http.StatusOK != resList.StatusCode() {
		return err
	}
	if amount != *resList.JSON200.TotalElements {
		return err
	}
	return nil
}

func InfraAPICheckInstances(ctx context.Context,
	apiClient *api.ClientWithResponses,
	filter *string,
	amount int,
) error {
	resList, err := apiClient.GetInstancesWithResponse(
		ctx,
		&api.GetInstancesParams{
			Filter: filter,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)

	if http.StatusOK != resList.StatusCode() {
		return err
	}
	if amount != *resList.JSON200.TotalElements {
		return err
	}
	return nil
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
		err := simClient.Create(ctx, enUUID, enCredentals, false, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func InfrastructureManagerAPIRegisterHosts(ctx context.Context,
	apiClient *api.ClientWithResponses,
	cfg *flags_test.TestConfig,
	enUUIDs []string,
) error {
	for i := 0; i < cfg.AmountEdgeNodes; i++ {
		enUUID := enUUIDs[i]
		enSerial := enUUID[:8]
		enName := "TestHost"

		hostUUID, err := uuid.Parse(enUUID)
		if err != nil {
			return err
		}

		// Register Hosts
		AutoOnboardTrue := true
		hostRegisterReq := api.HostRegisterInfo{
			Name:         &enName,
			Uuid:         &hostUUID,
			SerialNumber: &enSerial,
			AutoOnboard:  &AutoOnboardTrue,
		}
		zlog.Info().Msgf("Registering node %v", enUUID)
		hostRegister, err := apiClient.PostComputeHostsRegisterWithResponse(
			ctx,
			hostRegisterReq,
			utils.AddJWTtoTheHeader,
			utils.AddProjectIDtoTheHeader,
		)
		if err != nil {
			return err
		}
		if http.StatusCreated != hostRegister.StatusCode() {
			return fmt.Errorf("failed to register host %v", enUUID)
		}
	}
	return nil
}
