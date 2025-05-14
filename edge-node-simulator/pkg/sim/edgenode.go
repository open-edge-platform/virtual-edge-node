// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sim

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/magefile/mage/sh"

	ensimapi "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/api/ensim/v1"
	ensim_agents "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/agents"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	ensim_kc "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/keycloak"
	ensim_onboard "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/onboard"
)

const (
	statusChanSize = 10
)

var (
	onboardTimeout    = 120 * time.Second
	waitProvisionDone = 5 * time.Second
)

// UUID is an alias for string subscription UUID.
type UUID string

type EdgeNode struct {
	UUID
	cfg        *defs.Settings
	readyChan  chan bool
	termChan   chan bool
	statusChan chan *ensimapi.NodeStatus
	wg         *sync.WaitGroup
	status     sync.Map
	agents     *ensim_agents.Agents
}

func NewEdgeNode(cfg *defs.Settings) *EdgeNode {
	return &EdgeNode{
		UUID:       UUID(cfg.ENGUID),
		cfg:        cfg,
		readyChan:  make(chan bool, 1),
		termChan:   make(chan bool, 1),
		statusChan: make(chan *ensimapi.NodeStatus, statusChanSize),
		wg:         &sync.WaitGroup{},
		status:     sync.Map{},
	}
}

func (en *EdgeNode) SetAgentsStates(states map[ensim_agents.AgentType]ensim_agents.AgentState) {
	en.agents.SetDesiredStates(states)
}

func (en *EdgeNode) GetAgentsStates() ensim_agents.StateMap {
	return en.agents.GetStates()
}

func (en *EdgeNode) GetAgentsStatus() map[ensimapi.StatusSource]*ensimapi.NodeStatus {
	status := make(map[ensimapi.StatusSource]*ensimapi.NodeStatus)
	en.status.Range(func(key, value interface{}) bool {
		stat, ok := value.(*ensimapi.NodeStatus)
		if !ok {
			return false
		}
		source, ok := key.(ensimapi.StatusSource)
		if !ok {
			return false
		}
		status[source] = stat
		return true
	})
	return status
}

func (en *EdgeNode) createFolders() error {
	err := sh.RunV("mkdir", "-p", en.cfg.BaseFolder)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to create base folder %s", en.cfg.BaseFolder)
		return err
	}
	return nil
}

func (en *EdgeNode) deleteFolders() error {
	err := sh.RunV("rm", "-rf", en.cfg.BaseFolder)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to delete base folder %s", en.cfg.BaseFolder)
		return err
	}
	return nil
}

func (en *EdgeNode) StatsCollector() {
	zlog.Info().Msgf("Started stats collector %s", en.cfg.ENGUID)
	for {
		select {
		case <-en.termChan:
			zlog.Info().Msgf("Finished stats collector %s", en.cfg.ENGUID)
			en.statusChan = nil
			return
		case stats := <-en.statusChan:
			en.status.Store(stats.Source, stats)
		}
	}
}

func (en *EdgeNode) pushStatsEvent(
	source ensimapi.StatusSource,
	mode ensimapi.StatusMode,
	details string,
) {
	event := &ensimapi.NodeStatus{
		Source:  source,
		Mode:    mode,
		Details: details,
	}
	if en.statusChan != nil {
		en.statusChan <- event
	}
}

func (en *EdgeNode) startRequirements() error {
	go en.StatsCollector()

	err := en.createFolders()
	if err != nil {
		zlog.Error().Err(err).Msg("failed to create base folders")
		en.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_REQUIREMENTS,
			ensimapi.StatusMode_STATUS_MODE_FAILED,
			fmt.Sprintf("failed to create base folders: %s", err.Error()))
		return err
	}

	errDownloads := ensim_onboard.GetArtifacts(en.cfg)
	if errDownloads != nil {
		zlog.Error().Err(errDownloads).Msg("failed to download artifacts")
		en.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_REQUIREMENTS,
			ensimapi.StatusMode_STATUS_MODE_FAILED, errDownloads.Error())
	}
	en.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_REQUIREMENTS, ensimapi.StatusMode_STATUS_MODE_OK, "")
	return nil
}

func (en *EdgeNode) startOnboardProvision() error {
	ctx, cancel := context.WithTimeout(context.Background(), onboardTimeout)
	defer cancel()

	errRegister := ensim_onboard.RegisterProvisionEdgeNode(ctx, en.cfg)
	if errRegister != nil {
		zlog.Error().Err(errRegister).Msg("failed to register/provision edge node")
		en.pushStatsEvent(
			ensimapi.StatusSource_STATUS_SOURCE_ONBOARDED,
			ensimapi.StatusMode_STATUS_MODE_FAILED,
			errRegister.Error(),
		)
		return errRegister
	}

	errOnb := ensim_onboard.SouthOnboardNIO(ctx, en.cfg)
	if errOnb != nil {
		zlog.Error().Err(errOnb).Msg("failed to onboard")
		en.pushStatsEvent(
			ensimapi.StatusSource_STATUS_SOURCE_ONBOARDED,
			ensimapi.StatusMode_STATUS_MODE_FAILED,
			errOnb.Error(),
		)
		return errOnb
	}
	en.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_ONBOARDED, ensimapi.StatusMode_STATUS_MODE_OK, "NIO successful")

	errProv := ensim_onboard.SouthProvision(ctx, en.cfg)
	if errProv != nil {
		zlog.Error().Err(errProv).Msg("failed to provision")
		en.pushStatsEvent(
			ensimapi.StatusSource_STATUS_SOURCE_PROVISIONED,
			ensimapi.StatusMode_STATUS_MODE_FAILED,
			errProv.Error(),
		)
		return errProv
	}
	en.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_PROVISIONED, ensimapi.StatusMode_STATUS_MODE_OK, "")

	errCreds := ensim_onboard.SouthCredentials(ctx, en.cfg)
	if errCreds != nil {
		zlog.Error().Err(errCreds).Msg("failed to set credentials")
		en.pushStatsEvent(
			ensimapi.StatusSource_STATUS_SOURCE_CREDENTIALS,
			ensimapi.StatusMode_STATUS_MODE_FAILED,
			errCreds.Error(),
		)
		return errCreds
	}
	en.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_CREDENTIALS, ensimapi.StatusMode_STATUS_MODE_OK, "")

	return nil
}

func (en *EdgeNode) startAgents() error {
	authConf := ensim_kc.ConfigAuth{
		UUID:            en.cfg.ENGUID,
		CertCA:          en.cfg.CertCA,
		AccessTokenURL:  "keycloak." + en.cfg.OrchFQDN,
		RsTokenURL:      "token-provider." + en.cfg.OrchFQDN,
		AccessTokenPath: en.cfg.BaseFolder + "/tokens",
		ClientCredsPath: en.cfg.BaseFolder + "/client-credentials",
		TokenClients: []string{
			"node-agent",
			"hd-agent",
			"cluster-agent",
			"platform-update-agent",
			"platform-observability-agent",
			"platform-telemetry-agent",
			"prometheus",
			"license-agent",
		},
	}

	tokenMngr := ensim_kc.NewTokenManager(authConf)
	err := tokenMngr.Start(en.termChan, en.wg)
	if err != nil {
		zlog.Error().Err(err).Msg("failed to start token manager")
		en.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_TOKEN_MANAGER, ensimapi.StatusMode_STATUS_MODE_FAILED,
			fmt.Sprintf("failed to start token manager: %s", err.Error()))
		return err
	}

	if en.cfg.RunAgents {
		agents := ensim_agents.NewAgents(en.wg, en.readyChan, en.termChan, en.statusChan, en.cfg)
		err := agents.Start()
		if err != nil {
			zlog.Error().Err(err).Msg("failed to start simulated agents")
			en.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_SETUP, ensimapi.StatusMode_STATUS_MODE_FAILED,
				fmt.Sprintf("failed to start simulated agents: %s", err.Error()))
			return err
		}
		en.agents = agents
	}

	return nil
}

func (en *EdgeNode) Start() error {
	err := en.startRequirements()
	if err != nil {
		zlog.Error().Err(err).Msg("failed to start requirements")
		return err
	}
	err = en.startOnboardProvision()
	if err != nil {
		zlog.Error().Err(err).Msg("failed to start onboard/provision")
		return err
	}
	time.Sleep(waitProvisionDone)
	err = en.startAgents()
	if err != nil {
		zlog.Error().Err(err).Msg("failed to start agents")
		return err
	}
	if en.cfg.SetupTeardown {
		// Waits termChan to delete enic in Edge Infra API (host/instance)
		zlog.Info().Msg("Teardown Enabled")
		err := ensim_onboard.WaitTeardown(en.wg, en.termChan, en.cfg)
		if err != nil {
			zlog.Error().Err(err).Msg("failed to teardown")
			en.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_TEARDOWN, ensimapi.StatusMode_STATUS_MODE_FAILED,
				fmt.Sprintf("failed to teardown: %s", err.Error()))
		}
		en.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_TEARDOWN, ensimapi.StatusMode_STATUS_MODE_OK, "")
	}

	zlog.Info().Msgf("Edge Node Succefully onboarded/started %s", en.cfg.ENGUID)
	en.pushStatsEvent(ensimapi.StatusSource_STATUS_SOURCE_SETUP, ensimapi.StatusMode_STATUS_MODE_OK,
		"succefully onboarded/started")
	return nil
}

func (en *EdgeNode) Stop() error {
	close(en.termChan)
	zlog.Info().Msgf("Waiting edge node to stop: %s", en.cfg.ENGUID)
	en.wg.Wait()

	zlog.Info().Msgf("Deleting edge node folders: %s", en.cfg.ENGUID)
	err := en.deleteFolders()
	if err != nil {
		zlog.Error().Err(err).Msg("failed to delete edge node folders")
		return err
	}
	zlog.Info().Msgf("Edge Node succefully stopped: %s", en.cfg.ENGUID)
	return nil
}
