// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package onboard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/cenkalti/backoff"

	onboarding_pb "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/api/onboardingmgr/v1"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	ensim_kc "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/keycloak"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

var (
	tinkerTimeout = 11 * time.Minute // Waits for 11 minutes before timing out tinker worker.
	kcTimeout     = 60 * time.Second
)

// Defines constants for NIO - it will repeat at a max of 10 times with a backoff interval of 10 seconds.
const (
	backoffInterval = 10 * time.Second
	backoffRetries  = 10
)

// receiveFromStream receive a message from the stream.
func receiveFromStream(stream onboarding_pb.NonInteractiveOnboardingService_OnboardNodeStreamClient) (
	*onboarding_pb.OnboardNodeStreamResponse, error,
) {
	zlog.Info().Msgf("OnboardNodeStream: receiveFromStream")
	recv, err := stream.Recv()
	if errors.Is(err, io.EOF) {
		zlog.Info().Msgf("OnboardNodeStream client has closed the stream")
		return nil, io.EOF
	}
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("OnboardNodeStream error receiving from stream: %v", err)
		return nil, err
	}
	return recv, nil
}

func doNIO(
	ctx context.Context,
	kcToken string,
	cfg *defs.Settings,
) error {
	omAddress := fmt.Sprintf("onboarding-stream.%s:443", cfg.OrchFQDN)
	zlog.Info().Msgf("Connecting to Onboarding Manager %s", omAddress)
	conn, err := utils.Connect(omAddress, cfg.CertCAPath, kcToken)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to connect to Onboarding Manager %s", cfg.ENGUID)
		return err
	}
	defer conn.Close()
	client := onboarding_pb.NewNonInteractiveOnboardingServiceClient(conn)

	msgInit := &onboarding_pb.OnboardNodeStreamRequest{
		HostIp:    "192.168.1.1",
		Serialnum: cfg.ENSerial,
		Uuid:      cfg.ENGUID,
		MacId:     cfg.MACAddress,
	}

	zlog.Info().Msgf("Calling Onboarding Manager NodeStream: %v", msgInit)
	clientStream, err := client.OnboardNodeStream(ctx)
	if err != nil {
		return err
	}

	err = clientStream.Send(msgInit)
	if err != nil {
		return err
	}

	recv, err := receiveFromStream(clientStream)
	if err != nil {
		return err
	}

	status := recv.GetStatus()

	// codes.OK is the only valid status code for the initial message.
	if status.GetCode() != 0 {
		err = fmt.Errorf("failed to rcv stream msg %s", status)
		return err
	}

	if recv.GetNodeState() != onboarding_pb.OnboardNodeStreamResponse_NODE_STATE_ONBOARDED {
		err = fmt.Errorf("failed edge node state not onboarded: %s", recv.GetNodeState())
		zlog.Error().Err(err).Msgf("Edge node not onboarded yet")
		return err
	}

	clientID := recv.GetClientId()
	clientSecret := recv.GetClientSecret()
	projectID := recv.GetProjectId()
	zlog.Debug().Msgf("NIO in Onboarded state: clientID %s clientSecret %s projectID %s",
		clientID, clientSecret, projectID)
	return nil
}

// Non-interactive onboarding.
func SouthOnboardNIO(ctx context.Context, cfg *defs.Settings) error {
	zlog.Info().Msgf("Start NIOnboarding Edge Node %s", cfg.ENGUID)
	kcToken, err := ensim_kc.GetKeycloakToken(
		ctx,
		cfg.CertCA,
		cfg.OrchFQDN,
		cfg.EdgeOnboardUser,
		cfg.EdgeOnboardPass,
		defs.OrchKcClientID,
	)
	if err != nil {
		zlog.Err(err).Msgf("failed to get keycloak API token")
		return err
	}

	if err = backoff.Retry(func() error {
		return doNIO(ctx, kcToken, cfg)
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(backoffInterval), backoffRetries)); err != nil {
		zlog.Error().Err(err).Msgf("failed to south NIOnboard")
		return err
	}

	return nil
}

// SouthProvision fakes the tinker workflow execution, waiting for the edge node to be configured/provisioned.
func SouthProvision(_ context.Context, cfg *defs.Settings) error {
	// Make sure keycloak folders are created before tinker actions (enable write of clientID/clientSecret).
	err := ensim_kc.Folders(cfg)
	if err != nil {
		zlog.Err(err).Msgf("failed to create keycloak folders")
		return err
	}

	tinkWorker, err := NewTinkWorker(cfg.MACAddress, cfg)
	if err != nil {
		zlog.Err(err).Msgf("failed to instantiate tinker worker")
		return err
	}

	zlog.Info().Msgf("Edge node autoProvision %v", cfg.AutoProvision)
	// Makes sure the tinker worflow does not timeout while waiting for the edge node to be configured.
	// I.e., the tinker worker will be waiting indefinitely for workflows.
	tinkWorkerCtx := context.Background()
	if cfg.AutoProvision {
		// If autoProvision is enabled, the tinker worker will execute the workflow with a ctx timeout.
		tinkWorkerCtxTimeout, canceltinkWorkerCtx := context.WithTimeout(context.Background(), tinkerTimeout)
		tinkWorkerCtx = tinkWorkerCtxTimeout
		defer canceltinkWorkerCtx()
	}
	err = tinkWorker.ExecuteWorkflow(tinkWorkerCtx)
	if err != nil {
		zlog.Err(err).Msgf("failed to execute tinker workflow")
		return err
	}
	return nil
}

// SouthCredentials sets the keycloak access_token  for the edge node based on the clientID/clientSecret.
func SouthCredentials(_ context.Context, cfg *defs.Settings) error {
	// Make sure keycloak clientID/clientSecret are written by tinker actions, so access_token is retrieved.
	kcCtx, cancelkcCtx := context.WithTimeout(context.Background(), kcTimeout)
	defer cancelkcCtx()
	err := ensim_kc.SouthKeycloack(kcCtx, cfg)
	if err != nil {
		zlog.Err(err).Msgf("failed to set keycloak clientID/clientSecret")
		return err
	}
	return nil
}
