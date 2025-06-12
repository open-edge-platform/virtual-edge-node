// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package onboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	edgeinfraapi "github.com/open-edge-platform/infra-core/apiv2/v2/pkg/api/v2"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	ensim_kc "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/keycloak"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

const (
	// wait time for HTTP request.
	waitTime = 5 * time.Second
)

func httpPost(ctx context.Context, client *http.Client, url, token string,
	data []byte, responseHooker func(*http.Response) error,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
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
		err = fmt.Errorf("HTTP POST to %s failed, status: %s", url, resp.Status)
		return err
	}

	if responseHooker != nil {
		if err := responseHooker(resp); err != nil {
			return err
		}
	}

	return nil
}

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
		ps := &edgeinfraapi.ListHostsResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if len(ps.Hosts) == 0 {
			return fmt.Errorf("empty host result for uuid %s", uuid)
		}
		host := (ps.Hosts)[0]
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
	zlog.Info().Msgf("Deleting Host/Instance of edge node in Edge Infra %s project %s", cfg.ENGUID, cfg.Project)
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

func HTTPInfraOnboardNewInstance(instanceURL, token, hostID, osID string, client *http.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), waitTime)
	defer cancel()

	instKind := edgeinfraapi.INSTANCEKINDMETAL
	instanceName := "test-instance"
	sf := edgeinfraapi.SECURITYFEATURENONE
	instance := edgeinfraapi.InstanceResource{
		HostID:          &hostID,
		OsID:            &osID,
		Kind:            &instKind,
		Name:            &instanceName,
		SecurityFeature: &sf,
	}

	data, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("failed to marshal instance data: %w", err)
	}

	responseHooker := func(res *http.Response) error {
		if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to create instance, status: %s", res.Status)
		}
		return nil
	}

	if err := httpPost(ctx, client, instanceURL, token, data, responseHooker); err != nil {
		return fmt.Errorf("HTTP POST request failed: %w", err)
	}

	return nil
}

func HTTPInfraOnboardGetOSID(ctx context.Context, url, token string, client *http.Client) (string, error) {
	rCtx, cancel := context.WithTimeout(ctx, waitTime)
	defer cancel()

	osID := ""
	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		os := &edgeinfraapi.ListOperatingSystemsResponse{}
		err = json.Unmarshal(b, &os)
		if err != nil {
			return err
		}
		if len(os.OperatingSystemResources) == 0 {
			return fmt.Errorf("empty os resources")
		}
		for _, osr := range os.OperatingSystemResources {
			if *osr.ProfileName == "microvisor-nonrt" {
				osID = *osr.ResourceId
				zlog.Debug().Msgf("Found OS: %s", osID)
				break
			}
		}
		if osID == "" {
			return fmt.Errorf("microvisor-nonrt profile not found")
		}
		return nil
	}
	if err := httpGet(rCtx, client, url, token, responseHooker); err != nil {
		return osID, err
	}

	return osID, nil
}

func HTTPInfraOnboardNewRegisterHost(
	url, token string,
	client *http.Client,
	hostUUID uuid.UUID,
	autoOnboard bool,
) (*edgeinfraapi.HostResource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), waitTime)
	defer cancel()

	name := "host-" + hostUUID.String()[0:4]
	hostUUIDString := hostUUID.String()
	hostRegisterInfo := &edgeinfraapi.HostRegister{
		Name:        &name,
		Uuid:        &hostUUIDString,
		AutoOnboard: &autoOnboard,
	}

	data, err := json.Marshal(hostRegisterInfo)
	if err != nil {
		return nil, err
	}

	zlog.Debug().Msgf("HostRegisterInfo %s", data)
	var registeredHost edgeinfraapi.HostResource

	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		err = json.Unmarshal(b, &registeredHost)
		if err != nil {
			return err
		}
		return nil
	}

	zlog.Debug().Msgf("Sending POST request to %s with token %s", url, token)
	if err := httpPost(ctx, client, url, token, data, responseHooker); err != nil {
		zlog.Debug().Msgf("HTTP POST request failed: %v", err)
		return nil, err
	}

	return &registeredHost, nil
}

func HTTPInfraOnboardGetHostID(ctx context.Context, url, token string, client *http.Client, uuid string) (string, error) {
	rCtx, cancel := context.WithTimeout(ctx, waitTime)
	defer cancel()

	hostID := ""
	responseHooker := func(res *http.Response) error {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		ps := &edgeinfraapi.ListHostsResponse{}
		err = json.Unmarshal(b, &ps)
		if err != nil {
			return err
		}
		if len(ps.Hosts) == 0 {
			return fmt.Errorf("empty host result for uuid %s", uuid)
		}
		zlog.Debug().Msgf("HostResource %#v", ps)
		hostID = *(ps.Hosts)[0].ResourceId
		return nil
	}
	if err := httpGet(rCtx, client, fmt.Sprintf("%s?uuid=%s", url, uuid), token, responseHooker); err != nil {
		return hostID, err
	}

	return hostID, nil
}

func RegisterProvisionEdgeNode(ctx context.Context, cfg *defs.Settings) error {
	zlog.Info().Msgf("Registering/Provisioning Edge Node on %s (project: %s) with user %s and password %s",
		cfg.OrchFQDN, cfg.Project, cfg.EdgeAPIUser, cfg.EdgeAPIPass)

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

	apiBaseURLTemplate := "https://api.%s/v1/projects/%s"
	baseProjAPIURL := fmt.Sprintf(apiBaseURLTemplate, cfg.OrchFQDN, cfg.Project)
	hostRegURL := baseProjAPIURL + "/compute/hosts/register"
	hostURL := baseProjAPIURL + "/compute/hosts"
	instanceURL := baseProjAPIURL + "/compute/instances"
	osURL := baseProjAPIURL + "/compute/os"

	enUUID, err := uuid.Parse(cfg.ENGUID)
	if err != nil {
		return fmt.Errorf("error parsing edge node UUID: %w", err)
	}

	_, err = HTTPInfraOnboardNewRegisterHost(hostRegURL, apiToken, httpClient, enUUID, true)
	if err != nil {
		return fmt.Errorf("error registering edge node: %w", err)
	}

	hostID, err := HTTPInfraOnboardGetHostID(ctx, hostURL, apiToken, httpClient, cfg.ENGUID)
	if err != nil {
		return fmt.Errorf("error getting edge node resourceID: %w", err)
	}

	osID, err := HTTPInfraOnboardGetOSID(ctx, osURL, apiToken, httpClient)
	if err != nil {
		return fmt.Errorf("error getting OS Ubuntu resourceID: %w", err)
	}

	err = HTTPInfraOnboardNewInstance(instanceURL, apiToken, hostID, osID, httpClient)
	if err != nil {
		return fmt.Errorf("error provisioning edge node: %w", err)
	}

	zlog.Info().Msgf("Registered/Provisioned Edge Node on %s (project: %s) with user %s and password %s",
		cfg.OrchFQDN, cfg.Project, cfg.EdgeAPIUser, cfg.EdgeAPIPass)

	return nil
}
