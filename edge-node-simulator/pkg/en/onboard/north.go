// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package onboard

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	edgeinfraapi "github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	ensim_kc "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/keycloak"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

func apiSetup(ctx context.Context, cfg *defs.Settings) error {
	APIToken, err := ensim_kc.GetKeycloakToken(
		ctx,
		cfg.CertCA,
		cfg.OrchFQDN,
		cfg.EdgeAPIUser,
		cfg.EdgeAPIPass,
		defs.OrchKcClientID,
	)
	if err != nil {
		zlog.Err(err).Msgf("failed to get keycloak API token")
		return err
	}
	err = os.Setenv(jwtTokenEnvVar, APIToken)
	if err != nil {
		zlog.Err(err).Msgf("failed to set env jwtToken")
		return err
	}

	err = os.Setenv(projectIDEnvVar, cfg.Project)
	if err != nil {
		zlog.Err(err).Msgf("failed to set env projectID")
		return err
	}
	return nil
}

func deleteHostInstance(ctx context.Context, apiClient *edgeinfraapi.ClientWithResponses, hostID, instanceID string) error {
	zlog.Info().Msgf("Deleting Instance %s in Edge Infra", instanceID)
	resDeleteInstance, err := apiClient.DeleteInstancesInstanceIDWithResponse(
		ctx,
		instanceID,
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	if err != nil {
		return err
	}
	if resDeleteInstance.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("failed to delete Instance %s resources code %d", instanceID, resDeleteInstance.StatusCode())
	}
	zlog.Info().Msgf("Deleted Instance %s in Edge Infra", instanceID)

	zlog.Info().Msgf("Deleting Host %s in Edge Infra", hostID)
	resDeleteHost, err := apiClient.DeleteComputeHostsHostIDWithResponse(
		ctx,
		hostID,
		edgeinfraapi.HostOperationWithNote{},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	if err != nil {
		return err
	}
	if resDeleteHost.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("failed to delete Host %s resources code %d", hostID, resDeleteHost.StatusCode())
	}
	zlog.Info().Msgf("Deleted Host %s in Edge Infra", hostID)
	return nil
}

func Teardown(ctx context.Context, cfg *defs.Settings) error {
	zlog.Info().Msgf("Deleting Host/Instance of ENiC in Edge Infra %s project %s", cfg.ENGUID, cfg.Project)
	err := apiSetup(ctx, cfg)
	if err != nil {
		return err
	}

	httpClient, err := utils.GetClientWithCA(cfg.CertCA)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf(orchAPIURL, cfg.OrchFQDN)
	apiClient, err := edgeinfraapi.NewClientWithResponses(apiURL, edgeinfraapi.WithHTTPClient(httpClient))
	if err != nil {
		return err
	}

	byUUIDFilter := fmt.Sprintf(`uuid = %q`, cfg.ENGUID)
	resList, err := apiClient.GetComputeHostsWithResponse(
		ctx,
		&edgeinfraapi.GetComputeHostsParams{
			Filter: &byUUIDFilter,
		},
		utils.AddJWTtoTheHeader,
		utils.AddProjectIDtoTheHeader,
	)
	if err != nil {
		return err
	}

	if resList.StatusCode() != http.StatusOK {
		return fmt.Errorf("failed to retrieve Host with UUID %s status %d", cfg.ENGUID, resList.StatusCode())
	}

	if resList.JSON200.Hosts == nil {
		return fmt.Errorf("failed to retrieve valid list of Hosts with UUID %s", cfg.ENGUID)
	}

	if len(*resList.JSON200.Hosts) != 1 {
		return fmt.Errorf("failed to retrieve unique Host with UUID %s", cfg.ENGUID)
	}

	host := (*resList.JSON200.Hosts)[0]
	hostID := *host.ResourceId
	instanceID := *host.Instance.ResourceId

	err = deleteHostInstance(ctx, apiClient, hostID, instanceID)
	if err != nil {
		return err
	}
	zlog.Info().Msgf("Successfully deleted host/instance in Edge Infra for UUID %s", cfg.ENGUID)
	return nil
}

func WaitTeardown(wg *sync.WaitGroup, termChan chan bool, cfg *defs.Settings) error {
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-termChan
		err := Teardown(context.Background(), cfg)
		if err != nil {
			zlog.Error().Err(err).Msgf("Failed to teardown %s", cfg.ENGUID)
		}
	}()

	return nil
}
