// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package onboard

import (
	"fmt"
	"os/exec"

	"github.com/open-edge-platform/virtual-edge-node/edge-node-simulator/pkg/en/defs"
)

type AritfactType string

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

// URLArtifact downloads an artifact from the given URL and forwards the output to /dev/null.
func (a *Artifact) URLArtifact(url string) error {
	cmd := exec.Command("curl", "-o", "/dev/null", url)
	if err := cmd.Run(); err != nil {
		zlog.Error().Err(err).Msgf("Failed to download artifact from URL %s", url)
		return err
	}
	return nil
}

// DownloadArtifact downloads an artifact from the given URL and forwards the output to /dev/null.
func (a *Artifact) OrasArtifact(url string) error {
	cmd := exec.Command("oras", "-o", "/dev/null", url)
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
		if err := a.OrasArtifact(artifactURL); err != nil {
			return err
		}
	default:
		if err := a.URLArtifact(artifactURL); err != nil {
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

func artifactsAgent(baseURL string) []*Artifact {
	artifacts := []*Artifact{
		NewArtifact("caddy",
			baseURL, "%s/edge-orch/en/deb/caddy:%s", "agent", "1.0.0", ArtifactTypeAgent),
		NewArtifact("node-agent",
			baseURL, "%s/edge-orch/en/deb/node-agent:%s", "agent", "1.0.0", ArtifactTypeAgent),
		NewArtifact(
			"hardware-discovery-agent",
			baseURL,
			"%s/edge-orch/en/deb/hardware-discovery-agent:%s",
			"agent",
			"1.0.0",
			ArtifactTypeAgent,
		),
		NewArtifact("cluster-agent",
			baseURL, "%s/edge-orch/en/deb/cluster-agent:%s", "agent", "1.0.0", ArtifactTypeAgent),
		NewArtifact(
			"platform-observability-agent",
			baseURL,
			"%s/edge-orch/en/deb/platform-observability-agent:%s",
			"agent",
			"1.0.0",
			ArtifactTypeAgent,
		),
		NewArtifact("trtl", baseURL, "%s/edge-orch/en/deb/trtl:%s", "agent", "1.0.0", ArtifactTypeAgent),
		NewArtifact(
			"inbm-cloudadapter-agent",
			baseURL,
			"%s/edge-orch/en/deb/inbm-cloudadapter-agent:%s",
			"agent",
			"1.0.0",
			ArtifactTypeAgent,
		),
		NewArtifact(
			"inbm-configuration-agent",
			baseURL,
			"%s/edge-orch/en/deb/inbm-configuration-agent:%s",
			"agent",
			"1.0.0",
			ArtifactTypeAgent,
		),
		NewArtifact(
			"inbm-dispatcher-agent",
			baseURL,
			"%s/edge-orch/en/deb/inbm-dispatcher-agent:%s",
			"agent",
			"1.0.0",
			ArtifactTypeAgent,
		),
		NewArtifact(
			"inbm-telemetry-agent",
			baseURL,
			"%s/edge-orch/en/deb/inbm-telemetry-agent%s",
			"agent",
			"1.0.0",
			ArtifactTypeAgent,
		),
		NewArtifact(
			"inbm-diagnostic-agent",
			baseURL,
			"%s/edge-orch/en/deb/inbm-diagnostic-agent:%s",
			"agent",
			"1.0.0",
			ArtifactTypeAgent,
		),
		NewArtifact("mqtt",
			baseURL, "%s/edge-orch/en/deb/mqtt:%s", "agent", "1.0.0", ArtifactTypeAgent),
		NewArtifact("inbc-program",
			baseURL, "%s/edge-orch/en/deb/inbc-program:%s", "agent", "1.0.0", ArtifactTypeAgent),
		NewArtifact(
			"platform-update-agent",
			baseURL,
			"%s/edge-orch/en/deb/platform-update-agent%s",
			"agent",
			"1.0.0",
			ArtifactTypeAgent,
		),
		NewArtifact(
			"platform-telemetry-agent",
			baseURL,
			"%s/edge-orch/en/deb/platform-telemetry-agent:%s",
			"agent",
			"1.0.0",
			ArtifactTypeAgent,
		),
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

func NewArtifacts(cfg *defs.Settings) []*Artifact {
	artifacts := []*Artifact{}
	artifacts = append(artifacts, artifactsAgent(cfg.URLFilesRS)...)
	artifacts = append(artifacts, artifactsTinker(cfg.OrchFQDN)...)
	artifacts = append(artifacts, artifactsImage(cfg.URLFilesRS)...)
	artifacts = append(artifacts, artifactsTinkerAction(cfg.URLFilesRS, cfg.TinkerActionsVersion)...)
	return artifacts
}

func GetArtifacts(cfg *defs.Settings) error {
	if cfg.EnableDownloads {
		zlog.Info().Msg("Downloading artifacts")
		artifacts := NewArtifacts(cfg)
		if err := DownloadArtifacts(artifacts); err != nil {
			zlog.Error().Err(err).Msg("Failed to download artifacts")
			return err
		}
	}
	return nil
}
