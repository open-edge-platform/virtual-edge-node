// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
)

var zlog = logging.GetLogger("ensim.client")

var (
	errFatal error
	sigChan  = make(chan os.Signal, 1) // channel to handle any interrupt signals

	addressSimulator = flag.String(
		"addressSimulator",
		"",
		"gRPC address (ip:port) of the Edge Node simulator",
	)
	orchProject = flag.String(
		"project",
		"",
		"default project",
	)
	orchEdgeOnboardPasswd = flag.String(
		"onbPass",
		"", // update <onb-pass>
		"default password of orch keycloak onboard username",
	)
	orchEdgeOnboardUser = flag.String(
		"onbUser",
		"", // update <onb-user>
		"default onboard username of orch keycloak",
	)
	orchEdgeAPIPasswd = flag.String(
		"apiPass",
		"", // update <api-pass>
		"default password of orch keycloak API username",
	)
	orchEdgeAPIUser = flag.String(
		"apiUser",
		"", // update <api-user>
		"default API username of orch keycloak",
	)
	enableNIO = flag.Bool(
		"enableNIO",
		false,
		"enables edge node NIO by default",
	)
	enableTeardown = flag.Bool(
		"enableTeardown",
		true,
		"enables edge node Teardown (removal from InfrastructureManager on delete) by default",
	)
)

func main() {
	defer func() {
		if errFatal != nil {
			zlog.Fatal().Err(errFatal).Msg("failed to start Edge Node simulator client")
		}
	}()
	zlog.Info().Msg("Edge Node Simulator Client")
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())

	client, err := ensim.NewClient(ctx, *addressSimulator)
	if err != nil {
		zlog.Err(err).Msg("failed to create Edge Node sim client")
		errFatal = err
		return
	}
	defer client.Close()

	cliCfg := &ensim.CliCfg{
		Project:         *orchProject,
		OnboardUsername: *orchEdgeOnboardUser,
		OnboardPassword: *orchEdgeOnboardPasswd,
		APIUsername:     *orchEdgeAPIUser,
		APIPassword:     *orchEdgeAPIPasswd,
		EnableNIO:       *enableNIO,
		EnableTeardown:  *enableTeardown,
	}
	c := ensim.NewCli(ctx, client, cliCfg)
	_, err = c.PromptRoot()
	if err != nil {
		zlog.Err(err).Msg("failed to start simulator client CLI")
		errFatal = err
		return
	}

	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	s := <-sigChan
	zlog.Info().Msgf("Exiting Edge Node simulator client: received signal %s", s)
	cancel()
}
