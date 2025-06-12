// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package flags

import (
	"flag"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

var zlog = logging.GetLogger("flags")

type TestConfig struct {
	ClusterFQDN      string
	EdgeAPIUser      string
	EdgeAPIPass      string
	EdgeOnboardUser  string
	EdgeOnboardPass  string
	CAPath           string
	ENSimAddress     string
	Project          string
	AmountEdgeNodes  int
	BatchEdgeNodes   int
	DeployEdgeNodes  bool
	CreateOrgProject bool
	Cleanup          bool
}

func GetDefaultConfig() *TestConfig {
	return &TestConfig{
		ClusterFQDN:      "kind.internal",
		EdgeAPIUser:      "", // update <api-user>
		EdgeAPIPass:      "", // update <api-pass>
		EdgeOnboardUser:  "", // update <onb-user>
		EdgeOnboardPass:  "", // update <onb-pass>
		CAPath:           "",
		ENSimAddress:     "localhost:5001",
		Project:          "",
		AmountEdgeNodes:  1,
		BatchEdgeNodes:   1,
		DeployEdgeNodes:  false,
		CreateOrgProject: false,
		Cleanup:          false,
	}
}

var (
	defaultCfg = GetDefaultConfig()

	flagOrchestratorFQDN = flag.String(
		"clusterFQDN", defaultCfg.ClusterFQDN,
		"The orch cluster FQDN",
	)

	flagEdgeAPIUser = flag.String(
		"edgeAPIUser", defaultCfg.EdgeAPIUser,
		"The orch cluster EdgeAPIUser",
	)

	flagEdgeAPIPass = flag.String(
		"edgeAPIPass", defaultCfg.EdgeAPIPass,
		"The orch cluster EdgeAPIPass",
	)

	flagEdgeOnboardUser = flag.String(
		"edgeOnboardUser", defaultCfg.EdgeOnboardUser,
		"The orch cluster EdgeOnboardUser",
	)

	flagEdgeOnboardPass = flag.String(
		"edgeOnboardPass", defaultCfg.EdgeOnboardPass,
		"The orch cluster EdgeOnboardPass",
	)

	simAddress = flag.String(
		"simAddress",
		defaultCfg.ENSimAddress, "The gRPC address of the Infrastructure Manager simulator",
	)

	caPath = flag.String(
		"caFilepath",
		"", "The Infrastructure Manager cert CA file path",
	)

	project = flag.String(
		"project",
		defaultCfg.Project, "The project name",
	)

	amountEdgeNodes = flag.Int(
		"amountEdgeNodes",
		defaultCfg.AmountEdgeNodes, "The amount of edge nodes to be used in the tests",
	)

	batchEdgeNodes = flag.Int(
		"batchEdgeNodes",
		defaultCfg.BatchEdgeNodes, "The amount of edge nodes to be created in parallel in the tests",
	)

	deployEdgeNodes = flag.Bool(
		"deployEdgeNodes",
		defaultCfg.DeployEdgeNodes, "Flag to deploy edge nodes to execute tests",
	)

	createOrgProject = flag.Bool(
		"createOrgProject",
		defaultCfg.CreateOrgProject, "Flag to create org/project to execute tests",
	)

	cleanup = flag.Bool(
		"cleanup",
		defaultCfg.Cleanup, "Flag to perform cleanup of hosts/instances/schedules in Infrastructure Manager",
	)
)

func GetConfig() *TestConfig {
	flag.Parse()

	cfg := &TestConfig{
		ClusterFQDN:      *flagOrchestratorFQDN,
		EdgeAPIUser:      *flagEdgeAPIUser,
		EdgeAPIPass:      *flagEdgeAPIPass,
		EdgeOnboardUser:  *flagEdgeOnboardUser,
		EdgeOnboardPass:  *flagEdgeOnboardPass,
		ENSimAddress:     *simAddress,
		CAPath:           *caPath,
		Project:          *project,
		AmountEdgeNodes:  *amountEdgeNodes,
		BatchEdgeNodes:   *batchEdgeNodes,
		DeployEdgeNodes:  *deployEdgeNodes,
		CreateOrgProject: *createOrgProject,
		Cleanup:          *cleanup,
	}
	zlog.Info().Msgf("GetConfig %v", cfg)
	return cfg
}
