// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/magefile/mage/sh"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/oam"
	en_defs "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	ensim_onboard "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/onboard"
	ensim_utils "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

var zlog = logging.GetLogger("enic")

var (
	errFatal  error
	wg        = sync.WaitGroup{}        // waitgroup so main will wait for all go routines to exit cleanly
	readyChan = make(chan bool, 1)      // channel to signal the readiness.
	termChan  = make(chan bool, 1)      // channel to signal termination of main process.
	sigChan   = make(chan os.Signal, 1) // channel to handle any interrupt signals

	onboardTimeout       = 120 * time.Second
	defaultoamServerAddr = "0.0.0.0:5991"
	defaultBaseFolder    = "/tmp/enic"
	defaultOrchFQDN      = "kind.internal"
	defaultPasswd        = "" // update <password>
	defaultOnbUser       = "" // update <onb-user>
	defaultAPIUser       = "" // update <api-user>
)

var (
	orchFQDN = flag.String(
		"orchFQDN",
		defaultOrchFQDN,
		"FQDN of Orchestrator",
	)
	orchCAPath = flag.String(
		"orchCAPath",
		"",
		"Path of Orchestrator CA file",
	)
	baseFolder = flag.String(
		"baseFolder",
		defaultBaseFolder,
		"Path of folder to store edge node credentials/tokens",
	)
	oamServerAddr = flag.String(
		"oamServerAddress",
		defaultoamServerAddr,
		"default OAM server address",
	)
	orchProjectID = flag.String(
		"projectID",
		"",
		"default project ID",
	)
	orchEdgeOnboardPasswd = flag.String(
		"onbPass",
		defaultPasswd,
		"default password of orch keycloak onboard username",
	)
	orchEdgeOnboardUser = flag.String(
		"onbUser",
		defaultOnbUser,
		"default onboard username of orch keycloak",
	)
	orchEdgeAPIPasswd = flag.String(
		"apiPass",
		defaultPasswd,
		"default password of orch keycloak API username",
	)
	orchEdgeAPIUser = flag.String(
		"apiUser",
		defaultAPIUser,
		"default API username of orch keycloak",
	)
	enableNIO = flag.Bool(
		"enableNIO",
		false,
		"enables NIO in edge node",
	)
)

func setOAM(oamServerAddr string, termChan, readyChan chan bool, wg *sync.WaitGroup) {
	if oamServerAddr != "" {
		// Add oam grpc server
		wg.Add(1)
		go func() {
			// Disable tracing by default
			if err := oam.StartOamGrpcServer(termChan, readyChan, wg,
				oamServerAddr, false); err != nil {
				zlog.Fatal().Err(err).Msg("failed to start oam grpc server")
			}
		}()
	}
}

func GetNodeUUID() (string, error) {
	zlog.Info().Msg("Getting Node UUID")
	out, err := sh.Output("dmidecode", "-s", "system-uuid")
	if err != nil {
		return "", err
	}
	outUUID := strings.Trim(out, "\n")
	zlog.Info().Msgf("Node UUID %s", outUUID)
	return outUUID, nil
}

func GetNodeSerial() (string, error) {
	zlog.Info().Msg("Getting Node Serial Number")
	out, err := sh.Output("dmidecode", "-s", "system-serial-number")
	if err != nil {
		return "", err
	}
	outSerial := strings.Trim(out, "\n")
	zlog.Info().Msgf("Node Serial Number %s", outSerial)
	return outSerial, nil
}

func getCfg() (*en_defs.Settings, error) {
	enUUID, err := GetNodeUUID()
	if err != nil {
		zlog.Err(err).Msg("failed to get UUID")
		return nil, err
	}

	enSerial, err := GetNodeSerial()
	if err != nil {
		zlog.Err(err).Msg("failed to get serial")
		return nil, err
	}

	macAddress, err := ensim_utils.GetRandomMACAddress()
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to get random MAC address")
		return nil, err
	}

	// check if the values are empty or not
	if *orchFQDN == "" {
		zlog.Err(err).Msg("orchFQDN is mandatory parameter and it is empty")
		return nil, err
	}
	if *orchEdgeOnboardUser == "" {
		zlog.Err(err).Msg("orchEdgeOnboardUser is mandatory parameter and it is empty")
		return nil, err
	}
	if *orchEdgeOnboardPasswd == "" {
		zlog.Err(err).Msg("orchEdgeOnboardPasswd is mandatory parameter and it is empty")
		return nil, err
	}
	if *orchCAPath == "" {
		zlog.Err(err).Msg("orchCAPath is mandatory parameter and it is empty")
		return nil, err
	}

	certCA, err := ensim_utils.LoadFile(*orchCAPath)
	if err != nil {
		zlog.Err(err).Msg("failed to read certCA config file")
		return nil, err
	}

	cfg := &en_defs.Settings{
		ENGUID:          enUUID,
		ENSerial:        enSerial,
		OrchFQDN:        *orchFQDN,
		EdgeOnboardUser: *orchEdgeOnboardUser,
		EdgeOnboardPass: *orchEdgeOnboardPasswd,
		EdgeAPIUser:     *orchEdgeAPIUser,
		EdgeAPIPass:     *orchEdgeAPIPasswd,
		CertCA:          certCA,
		CertCAPath:      *orchCAPath,
		Project:         *orchProjectID,
		BaseFolder:      *baseFolder,
		MACAddress:      macAddress,
		OamServerAddr:   *oamServerAddr,
		NIOnboard:       *enableNIO,
		ENiC:            true,
	}
	zlog.Info().Msgf("Init cfg: %v", cfg)
	return cfg, err
}

func main() {
	zlog.Info().Msg("Onboarding Edge Node")
	flag.Parse()

	defer func() {
		if errFatal != nil {
			zlog.Fatal().Err(errFatal).Msg("failed to onboard edge-node")
		}
	}()

	cfg, err := getCfg()
	if err != nil {
		errFatal = err
		return
	}

	setOAM(cfg.OamServerAddr, termChan, readyChan, &wg)

	ctx, cancel := context.WithTimeout(context.Background(), onboardTimeout)
	defer cancel()

	if cfg.NIOnboard {
		err = ensim_onboard.SouthOnboardNIO(ctx, cfg)
		if err != nil {
			zlog.Error().Err(err).Msg("failed to onboard")
			errFatal = err
			return
		}
	} else {
		err = ensim_onboard.SouthOnboard(ctx, cfg)
		if err != nil {
			zlog.Error().Err(err).Msg("failed to onboard")
			errFatal = err
			return
		}
	}
	err = ensim_onboard.SouthProvision(ctx, cfg)
	if err != nil {
		zlog.Error().Err(err).Msg("failed to provision")
		errFatal = err
		return
	}
	err = ensim_onboard.SouthCredentials(ctx, cfg)
	if err != nil {
		zlog.Error().Err(err).Msg("failed to set credentials")
		errFatal = err
		return
	}

	zlog.Info().Msg("Edge Node Onboarded/Provisioned")
	readyChan <- true
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	zlog.Info().Msg("Exiting")
	close(termChan)

	// wait until oam server terminate
	wg.Wait()
}
