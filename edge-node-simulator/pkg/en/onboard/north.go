// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package onboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	edgeinfraapi "github.com/open-edge-platform/infra-core/api/pkg/api/v0"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	ensim_kc "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/keycloak"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

const (
	// wait time for HTTP request.
	waitTime = 5 * time.Second
)

func httpGet(ctx context.Context, client *http.Client, url, token string, responseHooker func(*http.Response) error) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP GET to %s failed, status: %s", url, resp.Status)
		return err
	}

	if responseHooker != nil {
		if err := responseHooker(resp); err != nil {
			return err
		}
	}

	return nil
}

func httpDelete(ctx context.Context, client *http.Client,
	url, token, resourceID string,
	responseHooker func(*http.Response) error,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/%s", url, resourceID), http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP Delete to %s failed, status: %s",
			fmt.Sprintf("%s/%s", url, resourceID), resp.Status)
		return err
	}

	if responseHooker != nil {
		if err := responseHooker(resp); err != nil {
			return err
		}
	}

	return nil
}

func HTTPInfraOnboardDelResource(ctx context.Context,
	client *http.Client,
	url, token, resourceID string,
) error {
	rCtx, cancel := context.WithTimeout(ctx, waitTime)
	defer cancel()

	if err := httpDelete(rCtx, client, url, token, resourceID, nil); err != nil {
		return err
	}

	return nil
}

func HTTPInfraOnboardGetHostAndInstance(ctx context.Context,
	client *http.Client,
	url, token, uuid string,
) (string, string, error) {
	rCtx, cancel := context.WithTimeout(ctx, waitTime)
	defer cancel()

	hostID := ""
	instanceID := ""
	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.HostsList{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if ps.Hosts == nil || len(*ps.Hosts) == 0 {
			return fmt.Errorf("empty host result for uuid %s", uuid)
		}
		host := (*ps.Hosts)[0]
		hostID = *host.ResourceId
		if host.Instance != nil && host.Instance.InstanceID != nil {
			instanceID = *host.Instance.InstanceID
		} else {
			return fmt.Errorf("instance not yet created for uuid %s", uuid)
		}
		return nil
	}

	if err := httpGet(rCtx, client, fmt.Sprintf("%s?uuid=%s", url, uuid), token, responseHooker); err != nil {
		return hostID, instanceID, err
	}

	return hostID, instanceID, nil
}

// cleanupHost is used to remove hosts that are created during the test.
func cleanupHost(ctx context.Context, hostURL, instanceURL, apiToken string, apiClient *http.Client, hostUUID string) error {
	hostID, instanceID, err := HTTPInfraOnboardGetHostAndInstance(ctx, apiClient, hostURL, apiToken, hostUUID)
	if err != nil {
		return err
	}
	zlog.Info().Msgf("Deleting Instance %s in Edge Infra", instanceID)
	err = HTTPInfraOnboardDelResource(ctx, apiClient, instanceURL, apiToken, instanceID)
	if err != nil {
		zlog.Error().Err(err).Msgf("Failed to delete instance %s in Edge Infra", instanceID)
		return err
	}
	zlog.Info().Msgf("Deleted Instance %s in Edge Infra", instanceID)

	zlog.Info().Msgf("Deleting Host %s in Edge Infra", hostID)
	err = HTTPInfraOnboardDelResource(ctx, apiClient, hostURL, apiToken, hostID)
	if err != nil {
		zlog.Error().Err(err).Msgf("Failed to delete host %s in Edge Infra", hostID)
		return err
	}
	zlog.Info().Msgf("Deleted Host %s in Edge Infra", hostID)
	return nil
}

func Teardown(ctx context.Context, cfg *defs.Settings) error {
	zlog.Info().Msgf("Deleting Host/Instance of ENiC in Edge Infra %s project %s", cfg.ENGUID, cfg.Project)
	apiToken, err := ensim_kc.GetKeycloakToken(
		ctx,
		cfg.CertCA,
		cfg.OrchFQDN,
		cfg.EdgeAPIUser,
		cfg.EdgeAPIPass,
		defs.OrchKcClientID,
	)
	if err != nil {
		return err
	}

	httpClient, err := utils.GetClientWithCA(cfg.CertCA)
	if err != nil {
		return err
	}

	hostURL := fmt.Sprintf("https://api.%s/v1/projects/%s/compute/hosts", cfg.OrchFQDN, cfg.Project)
	instanceURL := fmt.Sprintf("https://api.%s/v1/projects/%s/compute/instances", cfg.OrchFQDN, cfg.Project)
	err = cleanupHost(ctx, hostURL, instanceURL, apiToken, httpClient, cfg.ENGUID)
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
