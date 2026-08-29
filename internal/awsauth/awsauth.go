// Package awsauth is the only package that resolves AWS SDK configuration and
// credentials. It keeps the credential chain and Price List endpoint selection
// in one place.
package awsauth

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// priceListRegions are the only regions that host the AWS Price List Query API.
var priceListRegions = map[string]bool{
	"us-east-1":  true,
	"ap-south-1": true,
}

// Load resolves AWS configuration using the standard SDK credential chain
// (environment, shared config/credentials, named profiles, SSO, and
// container/instance roles). profile and regionOverride may be empty.
//
// Missing credentials are not reported here; the first API call surfaces them so
// that offline operations (for example `price-checker version`) never require
// credentials.
func Load(ctx context.Context, profile, regionOverride string) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if regionOverride != "" {
		opts = append(opts, config.WithRegion(regionOverride))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load AWS config: %w", err)
	}
	return cfg, nil
}

// PriceListRegion returns the region to send Price List API calls to. The Price
// List API is only available in us-east-1 and ap-south-1; if the caller's
// configured region is ap-south-1 that is used, otherwise us-east-1.
func PriceListRegion(cfg aws.Config) string {
	if priceListRegions[cfg.Region] {
		return cfg.Region
	}
	return "us-east-1"
}

// CredentialHelp is the actionable message shown when a Price List call fails
// authentication.
const CredentialHelp = "AWS credentials not found or invalid — configure them via " +
	"environment variables, a shared config/credentials file, a named profile (--profile), " +
	"AWS SSO, or an instance/container role. Set the pricing region with --aws-region."
