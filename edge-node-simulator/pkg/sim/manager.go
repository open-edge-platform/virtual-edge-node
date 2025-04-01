// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sim

import (
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/orch-library/go/pkg/northbound"
)

var zlog = logging.GetLogger("ifmsim")

type Manager interface {
	Start() error
	Stop() error
}

type manager struct {
	store  Store
	cfg    *Config
	server *northbound.Server
}

func NewManager(cfg *Config) (Manager, error) {
	store := NewStore()

	return &manager{
		cfg:    cfg,
		store:  store,
		server: nil,
	}, nil
}

func (m *manager) Start() error {
	zlog.Info().Msg("Starting Manager")
	err := m.startNorthbound(m.cfg)
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) Stop() error {
	zlog.Info().Msg("Stopping Manager")
	m.server.Stop()
	return nil
}

func (m *manager) startNorthbound(cfg *Config) error {
	m.server = northbound.NewServer(northbound.NewServerCfg(
		cfg.CAPath,
		cfg.KeyPath,
		cfg.CertPath,
		cfg.GRPCPort,
		true,
		northbound.SecurityConfig{}))

	m.server.AddService(NewIFMSimService(m.store, cfg))

	doneCh := make(chan error)
	go func() {
		err := m.server.Serve(func(started string) {
			zlog.Info().Msgf("Started NBI on %s", started)
			close(doneCh)
		})
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}
