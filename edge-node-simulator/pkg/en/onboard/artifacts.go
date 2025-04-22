// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package onboard

import (
	"fmt"
	"os/exec"

	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
	"gopkg.in/yaml.v3"
)

type AritfactType string

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
}

// Manifest
type Manifest struct {
	Metadata   string    `yaml:"metadata"`
	Repository string    `yaml:"repository"`
	Packages   []Package `yaml:"packages"`
}

type Package struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	OCIArtifact string `yaml:"ociArtifact"`
}

// URLArtifact downloads an artifact from the given URL and forwards the output to /dev/null.
func URLArtifact(url string) error {
	cmd := exec.Command("curl", "-o", "/dev/null", url)
	if err := cmd.Run(); err != nil {
		zlog.Error().Err(err).Msgf("Failed to download artifact from URL %s", url)
		return err
	}
	return nil
}

// DownloadArtifact downloads an artifact from the given URL and forwards the output to /dev/null.
func OrasArtifact(url, output string) error {
	cmd := exec.Command("oras", "-o", output, url)
	if err := cmd.Run(); err != nil {
		zlog.Error().Err(err).Msgf("Failed to download oras artifact from URL %s", url)
		return err
	}
	return nil
}

func (a *Artifact) ParseURL() (string, error) {
	var parsedURL string
	switch a.ArtifactType {
	case ArtifactTypeAgent:
		parsedURL = fmt.Sprintf(a.ArtifactURL, a.ArtifactBaseURL, a.ArtifactVersion)
	case ArtifactTypeTinker:
		parsedURL = fmt.Sprintf(a.ArtifactURL, a.ArtifactBaseURL)
	case ArtifactTypeImage:
		parsedURL = fmt.Sprintf(a.ArtifactURL, a.ArtifactBaseURL)
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
	switch a.ArtifactType {
	case ArtifactTypeAgent:
		if err := OrasArtifact(artifactURL, "/dev/null"); err != nil {
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
	artifactID, artifactBaseURL, artifactURL, artifactName, artifactVersion string,
	artifactType AritfactType,
) *Artifact {
	return &Artifact{
		ArtifactID:      artifactID,
		ArtifactBaseURL: artifactBaseURL,
		ArtifactURL:     artifactURL,
		ArtifactName:    artifactName,
		ArtifactVersion: artifactVersion,
		ArtifactType:    artifactType,
	}
}

func DownloadArtifacts(artifacts []*Artifact) error {
	for _, artifact := range artifacts {
		zlog.Debug().Msgf("Downloading %s from URL %s", artifact.ArtifactName, artifact.ArtifactURL)
		if err := artifact.Download(); err != nil {
			zlog.Error().Err(err).Msgf("Failed to download %s from URL %s", artifact.ArtifactName, artifact.ArtifactURL)
			return err
		}
	}
	return nil
}

func artifactsManifestAgent(baseURL, manifestVersion string) (map[string]string, error) {
	manifestFilePath := "/tmp/ena-manifest.yaml"
	manifestURL := fmt.Sprintf("%s/edge-orch/en/files/ena-manifest:%s", baseURL, manifestVersion)
	if err := OrasArtifact(manifestURL, manifestFilePath); err != nil {
		return nil, err
	}

	var manifestData Manifest
	if err := yaml.Unmarshal([]byte(manifestFilePath), &manifestData); err != nil {
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

func artifactsAgent(baseURL string, agentsVersions map[string]string) []*Artifact {
	artifacts := []*Artifact{}
	for _, agentName := range agentsNames {
		artifact := NewArtifact(agentName,
			baseURL, "%s/edge-orch/en/deb/"+agentName+":%s", agentName, agentsVersions[agentName], ArtifactTypeAgent)
		artifacts = append(artifacts, artifact)
	}
	return artifacts
}

func artifactsTinker(baseURL string) []*Artifact {
	artifacts := []*Artifact{
		NewArtifact(
			"iPXE binary",
			baseURL,
			"https://tinkerbell-nginx.%s/tink-stack/ipxe.efi",
			"image",
			"",
			ArtifactTypeTinker,
		),
		NewArtifact(
			"Boot ipxe",
			baseURL,
			"https://tinkerbell-nginx.%s/tink-stack/boot.ipxe",
			"image",
			"",
			ArtifactTypeTinker,
		),
		NewArtifact(
			"HookOS kernel",
			baseURL,
			"https://tinkerbell-nginx.%s/tink-stack/vmlinuz-x86_64",
			"image",
			"",
			ArtifactTypeTinker,
		),
		NewArtifact(
			"initramfs img",
			baseURL,
			"https://tinkerbell-nginx.%s/tink-stack/initramfs-x86_64",
			"image",
			"",
			ArtifactTypeTinker,
		),
	}
	return artifacts
}

func artifactsImage(baseURL string) []*Artifact {
	artifacts := []*Artifact{
		NewArtifact(
			"UbuntuOS",
			"",
			"https://cloud-images.ubuntu.com/releases/22.04/release-20250228/ubuntu-22.04-server-cloudimg-amd64.img",
			"image",
			"",
			ArtifactTypeImage,
		),
		NewArtifact(
			"TiberOS",
			baseURL,
			"https://%s/files-edge-orch/repository/TiberMicrovisor/",
			"image",
			"",
			ArtifactTypeImage,
		),
	}
	return artifacts
}

func artifactsTinkerAction(baseURL, tinkerVersion string) []*Artifact {
	artifacts := []*Artifact{
		NewArtifact(
			"securebootflag",
			baseURL,
			"%s/edge-orch/infra/tinker-actions/securebootflag:%s",
			"tinkeraction",
			tinkerVersion,
			ArtifactTypeTinkerAction,
		),
		NewArtifact(
			"qemu_nbd_image2disk",
			baseURL,
			"%s/edge-orch/infra/tinker-actions/qemu_nbd_image2disk:%s",
			"tinkeraction",
			tinkerVersion,
			ArtifactTypeTinkerAction,
		),
		NewArtifact(
			"image2disk",
			baseURL,
			"%s/edge-orch/infra/tinker-actions/image2disk:%s",
			"tinkeraction",
			tinkerVersion,
			ArtifactTypeTinkerAction,
		),
		NewArtifact(
			"writefile",
			baseURL,
			"%s/edge-orch/infra/tinker-actions/writefile:%s",
			"tinkeraction",
			tinkerVersion,
			ArtifactTypeTinkerAction,
		),
		NewArtifact(
			"efibootset",
			baseURL,
			"%s/edge-orch/infra/tinker-actions/efibootset:%s",
			"tinkeraction",
			tinkerVersion,
			ArtifactTypeTinkerAction,
		),
		NewArtifact(
			"kernelupgrd",
			baseURL,
			"%s/edge-orch/infra/tinker-actions/kernelupgrd:%s",
			"tinkeraction",
			tinkerVersion,
			ArtifactTypeTinkerAction,
		),
		NewArtifact(
			"tibermicrovisor_partition",
			baseURL,
			"%s/edge-orch/infra/tinker-actions/tibermicrovisor_partition:%s",
			"tinkeraction",
			tinkerVersion,
			ArtifactTypeTinkerAction,
		),
		NewArtifact(
			"fde",
			baseURL,
			"%s/edge-orch/infra/tinker-actions/fde:%s",
			"tinkeraction",
			tinkerVersion,
			ArtifactTypeTinkerAction,
		),
		NewArtifact(
			"cexec",
			baseURL,
			"%s/edge-orch/infra/tinker-actions/cexec:%s",
			"tinkeraction",
			tinkerVersion,
			ArtifactTypeTinkerAction,
		),
		NewArtifact(
			"erase_non_removable_disks",
			baseURL,
			"%s/edge-orch/infra/tinker-actions/erase_non_removable_disks:%s",
			"tinkeraction",
			tinkerVersion,
			ArtifactTypeTinkerAction,
		),
	}
	return artifacts
}

func NewArtifacts(cfg *defs.Settings, agentsVersions map[string]string) []*Artifact {
	artifacts := []*Artifact{}
	artifacts = append(artifacts, artifactsAgent(cfg.URLFilesRS, agentsVersions)...)
	artifacts = append(artifacts, artifactsTinker(cfg.OrchFQDN)...)
	artifacts = append(artifacts, artifactsImage(cfg.URLFilesRS)...)
	artifacts = append(artifacts, artifactsTinkerAction(cfg.URLFilesRS, cfg.TinkerActionsVersion)...)
	return artifacts
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
	}
	return nil
}
