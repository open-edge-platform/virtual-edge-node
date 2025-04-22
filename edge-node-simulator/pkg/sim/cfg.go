// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sim

import (
	"flag"
	"fmt"
	"math"
)

type Config struct {
	CAPath                string
	KeyPath               string
	CertPath              string
	GRPCPort              int16
	OrchFQDN              string
	OrchIP                string
	OrchCAPath            string
	BaseFolder            string
	OamServerAddr         string
	EnableDownloads       bool
	URLFilesRS            string
	TinkerActionsVersion  string
	AgentsManifestVersion string
}

var (
	caPath = flag.String(
		"caPath",
		"",
		"CA path for gRPC server",
	)
	keyPath = flag.String(
		"keyPath",
		"",
		"keyPath for gRPC server",
	)
	certPath = flag.String(
		"certPath",
		"",
		"certPath for gRPC server",
	)
	gRPCPort = flag.Int(
		"gRPCPort",
		defaultgRPCPort,
		"gRPCPort for server",
	)

	orchFQDN = flag.String(
		"orchFQDN",
		defaultOrchFQDN,
		"orchestrator FQDN",
	)

	orchIP = flag.String(
		"orchIP",
		"",
		"IP of orch",
	)
	orchCAPath = flag.String(
		"orchCAPath",
		"",
		"Path of orch CA file",
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
	enableDownloads = flag.Bool(
		"enableDownloads",
		false,
		"enable downloads of artifacts in the simulator",
	)
	urlFilesRS = flag.String(
		"urlFilesRS",
		defaultURLFilesRS,
		"URL of files for RS",
	)
	tinkerActionsVersion = flag.String(
		"tinkerActionsVersion",
		defaultTinkerActionsVersion,
		"Version of tinker actions",
	)
	agentsManifestVersion = flag.String(
		"agentsManifestVersion",
		defaultAgentsManifestVersion,
		"Version of agents manifest",
	)
)

// IntToint16 safely converts int to int16. This is needed for 64bit systems where int is defined as a 64bit integer.
// Returns an error when the value is out of the range.
func IntToInt16(i int) (int16, error) {
	if i < math.MinInt16 || i > math.MaxInt16 {
		return 0, fmt.Errorf("int value exceeds int16 range")
	}
	res := int16(i)
	if int(res) != i {
		zlog.InfraSec().InfraError("%#v of type int is out of range for int16", i).Msg("")
		return 0, fmt.Errorf("%#v of type int is out of range for int16", i)
	}
	return res, nil
}

func Cfg() (*Config, error) {
	flag.Parse()

	grpcPort, err := IntToInt16(*gRPCPort)
	if err != nil {
		zlog.InfraSec().InfraError("failed to convert gRPCPort to int16").Err(err).Msg("")
		return nil, err
	}

	cfg := &Config{
		CAPath:                *caPath,
		KeyPath:               *keyPath,
		CertPath:              *certPath,
		GRPCPort:              grpcPort,
		OrchFQDN:              *orchFQDN,
		OrchIP:                *orchIP,
		OrchCAPath:            *orchCAPath,
		BaseFolder:            *baseFolder,
		OamServerAddr:         *oamServerAddr,
		EnableDownloads:       *enableDownloads,
		URLFilesRS:            *urlFilesRS,
		TinkerActionsVersion:  *tinkerActionsVersion,
		AgentsManifestVersion: *agentsManifestVersion,
	}
	zlog.Info().Msgf("Loaded cfg: %v", cfg)
	return cfg, nil
}
