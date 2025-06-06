// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package onboard

import (
	"fmt"
	"os/exec"

	"gopkg.in/yaml.v3"

	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/utils"
)

type AritfactType string

var (
	manifestFileFolder = "/tmp/"
	manifestFilePath   = "/tmp/ena-manifest.yaml"
)

var agentsNames = []string{
	"cluster-agent",
	"hardware-discovery-agent",
	"node-agent",
	"platform-observability-agent",
	"platform-telemetry-agent",
	"platform-update-agent",
	"caddy",
	"inbc-program",
	"inbm-configuration-agent",
	"inbm-cloudadapter-agent",
	"inbm-diagnostic-agent",
	"inbm-dispatcher-agent",
	"inbm-telemetry-agent",
	"mqtt",
	"tpm-provision",
	"trtl",
}

var tinkerActionNames = []string{
	"securebootflag",
	"qemu_nbd_image2disk",
	"image2disk",
	"writefile",
	"efibootset",
	"kernelupgrd",
	"tibermicrovisor_partition",
	"fde",
	"cexec",
	"erase_non_removable_disks",
}

var tinkStackArtifactNames = []string{
	"ipxe.efi",
	"boot.ipxe",
	"vmlinuz-x86_64",
	"initramfs-x86_64",
}

const (
	ArtifactTypeAgent        AritfactType = "agent"
	ArtifactTypeTinker       AritfactType = "tinker"
	ArtifactTypeImage        AritfactType = "image"
	ArtifactTypeTinkerAction AritfactType = "tinkeraction"
)

type Artifact struct {
	ArtifactID      string       `json:"artifact_id"`
	ArtifactURL     string       `json:"artifact_url"`
	ArtifactBaseURL string       `json:"artifact_base_url"`
	ArtifactName    string       `json:"artifact_name"`
	ArtifactVersion string       `json:"artifact_version"`
	ArtifactType    AritfactType `json:"artifact_type"`
	ArtifactOutput  string       `json:"artifact_output"`
}

type Manifest struct {
	Packages []Package `yaml:"packages"`
}

type Package struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// URLArtifact downloads an artifact from the given URL and forwards the output to /dev/null.
func URLArtifact(url string) error {
	cmd := exec.Command("curl", "-k", "-o", "/dev/null", url)
	if err := cmd.Run(); err != nil {
		zlog.Error().Err(err).Msgf("Failed to download artifact from URL %s", url)
		return err
	}
	return nil
}

// DownloadArtifact downloads an artifact from the given URL and forwards the output to /dev/null.
func OrasArtifact(url, output string) error {
	cmd := exec.Command("oras", "pull", "-o", output, url)
	if err := cmd.Run(); err != nil {
		zlog.Error().Err(err).Msgf("Failed to download oras artifact from URL %s", url)
		return err
	}
	return nil
}

func (a *Artifact) ParseURL() (string, error) {
	var parsedURL string

	if a.ArtifactBaseURL == "" {
		zlog.Warn().Msgf("Base URL is empty for artifact %s", a.ArtifactName)
		return a.ArtifactURL, nil
	}
	switch a.ArtifactType {
	case ArtifactTypeAgent:
		parsedURL = fmt.Sprintf(a.ArtifactURL, a.ArtifactBaseURL, a.ArtifactVersion)
	case ArtifactTypeTinker:
		parsedURL = fmt.Sprintf(a.ArtifactURL, a.ArtifactBaseURL)
	case ArtifactTypeImage:
		parsedURL = fmt.Sprintf(a.ArtifactURL, a.ArtifactBaseURL, a.ArtifactVersion)
	case ArtifactTypeTinkerAction:
		parsedURL = fmt.Sprintf(a.ArtifactURL, a.ArtifactBaseURL, a.ArtifactVersion)
	default:
		err := fmt.Errorf("unsupported artifact type: %s", a.ArtifactType)
		zlog.Error().Err(err).Msgf("Failed to parse URL for artifact %s", a.ArtifactName)
		return "", err
	}
	return parsedURL, nil
}

func (a *Artifact) Download() error {
	artifactURL, err := a.ParseURL()
	if err != nil {
		zlog.Error().Err(err).Msgf("Failed to parse URL for artifact %s", a.ArtifactName)
		return err
	}
	zlog.Debug().Msgf("Downloading artifact %s from URL %s", a.ArtifactName, artifactURL)
	switch a.ArtifactType {
	case ArtifactTypeAgent:
		if err := OrasArtifact(artifactURL, a.ArtifactOutput); err != nil {
			return err
		}
	default:
		if err := URLArtifact(artifactURL); err != nil {
			return err
		}
	}
	return nil
}

func NewArtifact(
	artifactID, artifactBaseURL, artifactURL,
	artifactName, artifactVersion,
	artifactOutput string,
	artifactType AritfactType,
) *Artifact {
	return &Artifact{
		ArtifactID:      artifactID,
		ArtifactBaseURL: artifactBaseURL,
		ArtifactURL:     artifactURL,
		ArtifactName:    artifactName,
		ArtifactVersion: artifactVersion,
		ArtifactType:    artifactType,
		ArtifactOutput:  artifactOutput,
	}
}

func DownloadArtifacts(artifacts []*Artifact) error {
	for _, artifact := range artifacts {
		if err := artifact.Download(); err != nil {
			zlog.Error().Err(err).Msgf("Failed to download %s from URL %s", artifact.ArtifactName, artifact.ArtifactURL)
			return err
		}
	}
	return nil
}

func artifactsManifestAgent(baseURL, manifestVersion string) (map[string]string, error) {
	manifestURL := fmt.Sprintf("%s/edge-orch/en/files/ena-manifest:%s", baseURL, manifestVersion)
	if err := OrasArtifact(manifestURL, manifestFileFolder); err != nil {
		return nil, err
	}

	// Read the manifest file
	manifestRawData, err := utils.LoadFile(manifestFilePath)
	if err != nil {
		zlog.Error().Err(err).Msgf("Failed to read manifest file %s", manifestFilePath)
		return nil, err
	}

	var manifestData Manifest
	if err := yaml.Unmarshal([]byte(manifestRawData), &manifestData); err != nil {
		return nil, err
	}

	artifacts := make(map[string]string)
	for _, pkg := range manifestData.Packages {
		artifacts[pkg.Name] = pkg.Version
	}

	for _, agentName := range agentsNames {
		if _, ok := artifacts[agentName]; !ok {
			zlog.Error().Msgf("Agent %s not found in manifest", agentName)
			return nil, fmt.Errorf("agent %s not found in manifest", agentName)
		}
	}
	return artifacts, nil
}

func artifactsAgent(baseURL, outputDir string, agentsVersions map[string]string) []*Artifact {
	artifacts := []*Artifact{}
	for _, agentName := range agentsNames {
		artifact := NewArtifact(agentName,
			baseURL, "%s/edge-orch/en/deb/"+agentName+":%s",
			agentName, agentsVersions[agentName],
			outputDir,
			ArtifactTypeAgent)
		artifacts = append(artifacts, artifact)
	}
	return artifacts
}

func artifactsTinker(baseURL string) []*Artifact {
	artifacts := []*Artifact{}
	for _, artifactName := range tinkStackArtifactNames {
		artifact := NewArtifact(
			artifactName,
			baseURL,
			"https://tinkerbell-nginx.%s/tink-stack/"+artifactName,
			artifactName,
			"",
			"",
			ArtifactTypeTinker,
		)
		artifacts = append(artifacts, artifact)
	}
	return artifacts
}

func artifactsImage(baseURL, tiberOSVersion string) []*Artifact {
	artifacts := []*Artifact{
		NewArtifact(
			"UbuntuOS",
			"",
			"https://cloud-images.ubuntu.com/releases/22.04/release-20250228/ubuntu-22.04-server-cloudimg-amd64.img",
			"image",
			"",
			"",
			ArtifactTypeImage,
		),
		NewArtifact(
			"TiberOS",
			baseURL,
			"https://%s/files-edge-orch/repository/TiberMicrovisor/TiberMicrovisor_nonRT/tiber-readonly-%s",
			"image",
			tiberOSVersion,
			"",
			ArtifactTypeImage,
		),
	}
	return artifacts
}

func artifactsTinkerAction(baseURL, tinkerVersion string) []*Artifact {
	artifacts := []*Artifact{}
	for _, actionName := range tinkerActionNames {
		artifact := NewArtifact(
			actionName,
			baseURL,
			"%s/edge-orch/infra/tinker-actions/"+actionName+":%s",
			actionName,
			tinkerVersion,
			"",
			ArtifactTypeTinkerAction,
		)
		artifacts = append(artifacts, artifact)
	}
	return artifacts
}

func NewArtifacts(cfg *defs.Settings, agentsVersions map[string]string) []*Artifact {
	artifacts := []*Artifact{}
	artifacts = append(artifacts, artifactsAgent(cfg.URLFilesRS, cfg.BaseFolder, agentsVersions)...)
	artifacts = append(artifacts, artifactsTinker(cfg.OrchFQDN)...)
	artifacts = append(artifacts, artifactsImage(cfg.URLFilesRS, cfg.TiberOSVersion)...)
	artifacts = append(artifacts, artifactsTinkerAction(cfg.URLFilesRS, cfg.TinkerActionsVersion)...)
	return artifacts
}

func cleanArtifacts(outputDirs []string) error {
	for _, dir := range outputDirs {
		zlog.Debug().Msgf("Cleaning artifact %s", dir)
		if err := exec.Command("rm", "-rf", dir).Run(); err != nil {
			zlog.Error().Err(err).Msgf("Failed to clean artifacts folder %s", dir)
			return err
		}
	}
	return nil
}

func GetArtifacts(cfg *defs.Settings) error {
	if cfg.EnableDownloads {
		zlog.Info().Msg("Downloading ENA manifest")
		agentsVersions, err := artifactsManifestAgent(cfg.URLFilesRS, cfg.AgentsManifestVersion)
		if err != nil {
			zlog.Error().Err(err).Msg("Failed to download ENA manifest")
			return err
		}

		zlog.Info().Msg("Downloading artifacts")
		artifacts := NewArtifacts(cfg, agentsVersions)
		if err := DownloadArtifacts(artifacts); err != nil {
			zlog.Error().Err(err).Msg("Failed to download artifacts")
			return err
		}

		zlog.Info().Msg("Cleaning artifacts")
		outputDirs := []string{
			cfg.BaseFolder + "/*.deb",
		}
		if err := cleanArtifacts(outputDirs); err != nil {
			zlog.Error().Err(err).Msg("Failed to clean artifacts")
			return err
		}
	}
	return nil
}
