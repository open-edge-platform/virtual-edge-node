// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package utils //nolint:revive // to be refactored in future

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"

	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	JWTTokenEnvVar  = "JWT_TOKEN"
	ProjectIDEnvVar = "PROJECT_ID"
	macLength       = 6
)

const (
	local     = 0b10
	multicast = 0b1
)

var (
	nodeUUID   = uuid.New().String()
	nodeSerial = nodeUUID[:8]
	zlog       = logging.GetLogger("utils")
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

func GetAuthContext(ctx context.Context, tokenPath string) (context.Context, error) {
	tBytes, err := LoadFile(tokenPath)
	if err != nil {
		zlog.Err(err).Msgf("failed to load token from file %s", tokenPath)
		return nil, err
	}
	tString := fmt.Sprintf("Bearer %s", tBytes)
	header := metadata.New(map[string]string{"authorization": strings.TrimSpace(tString)})

	return metadata.NewOutgoingContext(ctx, header), nil
}

func GetNodeUUID() (string, error) {
	zlog.Info().Msg("Getting Node UUID")
	outUUID := nodeUUID
	zlog.Info().Msgf("Node UUID %s", outUUID)
	return outUUID, nil
}

func GetNodeSerial() (string, error) {
	zlog.Info().Msg("Getting Node Serial Number")
	outSerial := nodeSerial
	zlog.Info().Msgf("Node Serial Number %s", outSerial)
	return outSerial, nil
}

//nolint:gosec // InsecureSkipVerify: true, as we are using CA cert
func GetClientWithCA(caCert string) (*http.Client, error) {
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM([]byte(caCert))
	if !ok {
		err := fmt.Errorf("failed to parse CA cert into http client")
		return nil, err
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return &http.Client{
		Transport: transport,
	}, nil
}

// Connect creates a gRPC Connection to a server.
//
//nolint:gosec // InsecureSkipVerify: true, as we are using CA cert and token
func Connect(
	address string,
	caPath, token string,
) (*grpc.ClientConn, error) {
	var conn *grpc.ClientConn

	if caPath == "" || token == "" {
		err := fmt.Errorf("CaCertPath %s or token %s were not provided",
			caPath, token,
		)
		zlog.Err(err).Msg("error checking paths of cert/token")
		return nil, err
	}
	// setting secure gRPC Connection
	oauthToken := &oauth2.Token{
		AccessToken: token,
	}
	perRPC := oauth.TokenSource{TokenSource: oauth2.StaticTokenSource(oauthToken)}
	creds, err := credentials.NewClientTLSFromFile(caPath, address)
	if err != nil {
		log.Fatalf("failed to load credentials: %v", err)
	}

	if err != nil {
		zlog.Err(err).Msg("error in handle cert paths")
		return nil, err
	}

	roots, err := x509.SystemCertPool()
	if err != nil {
		zlog.Err(err).Msg("failed to get system certificate pool")
		return nil, err
	}

	config := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		RootCAs:            roots,
	}
	opts := []grpc.DialOption{
		// In addition to the following grpc.DialOption, callers may also use
		// the grpc.CallOption grpc.PerRPCCredentials with the RPC invocation
		// itself.
		// See: https://godoc.org/google.golang.org/grpc#PerRPCCredentials
		grpc.WithPerRPCCredentials(perRPC),
		// oauth.TokenSource requires the configuration of transport
		// credentials.
		grpc.WithTransportCredentials(creds),
		grpc.WithTransportCredentials(credentials.NewTLS(config)),
	}

	conn, err = grpc.NewClient(address, opts...)
	if err != nil {
		zlog.Info().Msgf("Unable to dial Connection to client address %s\n", address)
		zlog.Err(err).Msg("error Connecting to server")
		return nil, err
	}
	return conn, nil
}

func AddJWTtoTheHeader(_ context.Context, req *http.Request) error {
	jwtTokenStr, ok := os.LookupEnv(JWTTokenEnvVar)
	if !ok {
		return fmt.Errorf("can't find a \"JWT_TOKEN\" variable, please set it in your environment")
	}
	req.Header.Add("authorization", "Bearer "+jwtTokenStr)
	return nil
}

func AddProjectIDtoTheHeader(_ context.Context, req *http.Request) error {
	projectIDStr, ok := os.LookupEnv(ProjectIDEnvVar)
	if !ok {
		return fmt.Errorf("can't find a \"%s\" variable, please set it in your environment", ProjectIDEnvVar)
	}
	req.Header.Set("ActiveProjectID", projectIDStr)
	return nil
}

func GetRandomMACAddress() (string, error) {
	buf := make([]byte, macLength)
	_, err := rand.Read(buf)
	if err != nil {
		zlog.Error().Err(err).Msgf("failed to get rand for random MAC address")
		return "", err
	}
	// clear multicast bit (&^), ensure local bit (|)
	buf[0] = buf[0]&^multicast | local
	hwAddr := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])
	zlog.Info().Msgf("GetRandomMACAddress: %s", hwAddr)
	return hwAddr, nil
}
