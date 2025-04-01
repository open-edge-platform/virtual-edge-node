// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package sim

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/cert"
)

func LoadFile(filePath string) (string, error) {
	dirFile, err := filepath.Abs(filePath)
	if err != nil {
		zlog.Err(err).Msgf("failed LoadFile, filepath unexistent %s", filePath)
		return "", err
	}

	dataBytes, err := os.ReadFile(dirFile)
	if err != nil {
		zlog.Err(err).Msgf("failed to read file %s", dirFile)
		return "", err
	}

	dataStr := string(dataBytes)
	return dataStr, nil
}

// connect creates a gRPC connection to a server.
func Connect(
	_ context.Context,
	address string,
	caPath, certPath, keyPath string,
	insec bool,
	opts ...grpc.DialOption,
) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn

	if insec {
		dialOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
		opts = append(opts, dialOpt)
	} else {
		if caPath == "" || certPath == "" || keyPath == "" {
			err := fmt.Errorf("CaCertPath %s or TlsCerPath %s or TlsKeyPath %s were not provided",
				caPath, certPath, keyPath,
			)
			zlog.Fatal().Err(err).Msgf("CaCertPath %s or TlsCerPath %s or TlsKeyPath %s were not provided\n",
				caPath, certPath, keyPath,
			)
			return nil, err
		}
		// setting secure gRPC connection
		creds, err := cert.HandleCertPaths(caPath, keyPath, certPath, true)
		if err != nil {
			zlog.Fatal().Err(err).Msgf("an error occurred while loading credentials to server %v, %v, %v: %v\n",
				caPath, certPath, keyPath, err,
			)
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	}

	// if testing, use a bufconn, otherwise TCP
	var err error
	if address == "bufconn" {
		conn, err = grpc.NewClient("", opts...)
	} else {
		conn, err = grpc.NewClient(address, opts...)
	}
	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("Unable to dial connection to inventory client address %s", address)
		return nil, err
	}
	return conn, nil
}
