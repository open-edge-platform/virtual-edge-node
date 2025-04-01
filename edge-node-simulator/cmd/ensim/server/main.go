// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/oam"
	_ "github.com/open-edge-platform/infra-core/inventory/v2/pkg/perf"
	ensim "github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/sim"
)

var zlog = logging.GetLogger("ensim.server")

var (
	errFatal  error
	wg        = sync.WaitGroup{}        // waitgroup so main will wait for all go routines to exit cleanly
	readyChan = make(chan bool, 1)      // channel to signal the readiness.
	termChan  = make(chan bool, 1)      // channel to signal termination of main process.
	sigChan   = make(chan os.Signal, 1) // channel to handle any interrupt signals
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

func main() {
	zlog.Info().Msg("Edge Node Simulator")

	defer func() {
		if errFatal != nil {
			zlog.Fatal().Err(errFatal).Msg("failed to start Edge Node simulator")
		}
	}()

	cfg, err := ensim.Cfg()
	if err != nil {
		zlog.Err(err).Msg("failed to get config")
		errFatal = err
		return
	}
	setOAM(cfg.OamServerAddr, termChan, readyChan, &wg)

	mngr, err := ensim.NewManager(cfg)
	if err != nil {
		zlog.Err(err).Msg("failed to create manager")
		errFatal = err
		return
	}

	err = mngr.Start()
	if err != nil {
		zlog.Err(err).Msg("failed to start manager")
		errFatal = err
		return
	}
	readyChan <- true

	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	s := <-sigChan
	zlog.Info().Msgf("Exiting Edge Node simulator: received signal %s", s)
	mngr.Stop()
	close(termChan)

	// wait until agents / oam server / teardown terminates.
	wg.Wait()
}
