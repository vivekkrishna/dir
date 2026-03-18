// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// isECRRegistry returns true if the address is an ECR endpoint.
func isECRRegistry(address string) bool {
	return strings.Contains(address, ".dkr.ecr.")
}

// ecrTokenCache caches ECR authorization tokens with expiry.
type ecrTokenCache struct {
	mu       sync.RWMutex
	username string
	password string
	expiry   time.Time
}

// newECRAuthClient creates an ORAS auth.Client that uses
// AWS SDK GetAuthorizationToken for ECR authentication.
// Tokens are cached in-process and refreshed before the
// 12-hour expiry window.
func newECRAuthClient(registryAddress string) (*auth.Client, error) {
	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	ecrClient := ecr.NewFromConfig(awsCfg)
	cache := &ecrTokenCache{}

	credentialFunc := func(ctx context.Context, hostport string) (auth.Credential, error) {
		cache.mu.RLock()
		if time.Now().Before(cache.expiry.Add(-10 * time.Minute)) {
			cred := auth.Credential{Username: cache.username, Password: cache.password}
			cache.mu.RUnlock()
			return cred, nil
		}
		cache.mu.RUnlock()

		// Refresh token
		output, err := ecrClient.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
		if err != nil {
			return auth.Credential{}, fmt.Errorf(
				"ECR GetAuthorizationToken failed for %s: %w", registryAddress, err,
			)
		}
		if len(output.AuthorizationData) == 0 {
			return auth.Credential{}, fmt.Errorf("no authorization data from ECR")
		}

		token := output.AuthorizationData[0]
		decoded, err := base64.StdEncoding.DecodeString(*token.AuthorizationToken)
		if err != nil {
			return auth.Credential{}, fmt.Errorf("failed to decode ECR token: %w", err)
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return auth.Credential{}, fmt.Errorf("unexpected ECR token format")
		}

		cache.mu.Lock()
		cache.username = parts[0]
		cache.password = parts[1]
		cache.expiry = *token.ExpiresAt
		cache.mu.Unlock()

		return auth.Credential{Username: parts[0], Password: parts[1]}, nil
	}

	return &auth.Client{
		Client:     retry.DefaultClient,
		Header:     http.Header{"User-Agent": {"dir-client"}},
		Cache:      auth.DefaultCache,
		Credential: credentialFunc,
	}, nil
}
