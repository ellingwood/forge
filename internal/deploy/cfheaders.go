package deploy

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

const responseHeadersPolicyName = "forge-security-headers"

// ResponseHeadersConfig holds the security header values to apply via a
// CloudFront response headers policy.
type ResponseHeadersConfig struct {
	CSP                 string // full Content-Security-Policy header value
	HSTSMaxAge          int    // HSTS max-age in seconds
	HSTSSubDomains      bool   // HSTS includeSubDomains directive
	HSTSPreload         bool   // HSTS preload directive
	XContentTypeNosniff bool   // X-Content-Type-Options: nosniff
	XFrameOptions       string // Must be "DENY" or "SAMEORIGIN"
	ReferrerPolicy      string // Must be a valid Referrer-Policy value (e.g., "strict-origin-when-cross-origin")
}

// cfHeadersAPI is the subset of the CloudFront SDK client used by
// AWSCloudFrontHeadersPolicyClient.
type cfHeadersAPI interface {
	CreateResponseHeadersPolicy(ctx context.Context, params *cloudfront.CreateResponseHeadersPolicyInput, optFns ...func(*cloudfront.Options)) (*cloudfront.CreateResponseHeadersPolicyOutput, error)
	GetResponseHeadersPolicy(ctx context.Context, params *cloudfront.GetResponseHeadersPolicyInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetResponseHeadersPolicyOutput, error)
	UpdateResponseHeadersPolicy(ctx context.Context, params *cloudfront.UpdateResponseHeadersPolicyInput, optFns ...func(*cloudfront.Options)) (*cloudfront.UpdateResponseHeadersPolicyOutput, error)
	ListResponseHeadersPolicies(ctx context.Context, params *cloudfront.ListResponseHeadersPoliciesInput, optFns ...func(*cloudfront.Options)) (*cloudfront.ListResponseHeadersPoliciesOutput, error)
	GetDistributionConfig(ctx context.Context, params *cloudfront.GetDistributionConfigInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionConfigOutput, error)
	UpdateDistribution(ctx context.Context, params *cloudfront.UpdateDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.UpdateDistributionOutput, error)
}

// AWSCloudFrontHeadersPolicyClient implements CloudFrontHeadersPolicyClient
// using the AWS SDK v2.
type AWSCloudFrontHeadersPolicyClient struct {
	client cfHeadersAPI
}

// NewAWSCloudFrontHeadersPolicyClient creates a new AWSCloudFrontHeadersPolicyClient.
func NewAWSCloudFrontHeadersPolicyClient(client cfHeadersAPI) *AWSCloudFrontHeadersPolicyClient {
	return &AWSCloudFrontHeadersPolicyClient{client: client}
}

// EnsureResponseHeadersPolicy creates or updates a CloudFront response headers
// policy named "forge-security-headers" and associates it with the given
// distribution's default cache behavior. All mutations are idempotent.
func (c *AWSCloudFrontHeadersPolicyClient) EnsureResponseHeadersPolicy(
	ctx context.Context, distributionID string, cfg ResponseHeadersConfig,
) error {
	// Step 1: Build the policy config.
	policyCfg := buildResponseHeadersPolicyConfig(cfg)

	// Step 2: Create or update the policy to get its ID.
	policyID, err := c.ensurePolicy(ctx, policyCfg)
	if err != nil {
		return fmt.Errorf("ensuring response headers policy %q: %w", responseHeadersPolicyName, err)
	}

	// Step 3: Associate the policy with the distribution.
	if err := c.associatePolicyWithDistribution(ctx, distributionID, policyID); err != nil {
		return fmt.Errorf("associating response headers policy with distribution %q: %w", distributionID, err)
	}

	return nil
}

// buildResponseHeadersPolicyConfig constructs the CloudFront SDK policy config
// from our domain-level ResponseHeadersConfig.
func buildResponseHeadersPolicyConfig(cfg ResponseHeadersConfig) *cftypes.ResponseHeadersPolicyConfig {
	policyCfg := &cftypes.ResponseHeadersPolicyConfig{
		Name:    aws.String(responseHeadersPolicyName),
		Comment: aws.String("Forge security headers: CSP, HSTS, X-Content-Type-Options, X-Frame-Options, Referrer-Policy"),
		SecurityHeadersConfig: &cftypes.ResponseHeadersPolicySecurityHeadersConfig{},
	}

	sh := policyCfg.SecurityHeadersConfig

	// Content-Security-Policy
	if cfg.CSP != "" {
		sh.ContentSecurityPolicy = &cftypes.ResponseHeadersPolicyContentSecurityPolicy{
			ContentSecurityPolicy: aws.String(cfg.CSP),
			Override:              aws.Bool(true),
		}
	}

	// Strict-Transport-Security
	if cfg.HSTSMaxAge > 0 {
		sh.StrictTransportSecurity = &cftypes.ResponseHeadersPolicyStrictTransportSecurity{
			AccessControlMaxAgeSec: aws.Int32(int32(cfg.HSTSMaxAge)),
			IncludeSubdomains:      aws.Bool(cfg.HSTSSubDomains),
			Preload:                aws.Bool(cfg.HSTSPreload),
			Override:               aws.Bool(true),
		}
	}

	// X-Content-Type-Options
	if cfg.XContentTypeNosniff {
		sh.ContentTypeOptions = &cftypes.ResponseHeadersPolicyContentTypeOptions{
			Override: aws.Bool(true),
		}
	}

	// X-Frame-Options
	if cfg.XFrameOptions != "" {
		sh.FrameOptions = &cftypes.ResponseHeadersPolicyFrameOptions{
			FrameOption: cftypes.FrameOptionsList(cfg.XFrameOptions),
			Override:    aws.Bool(true),
		}
	}

	// Referrer-Policy
	if cfg.ReferrerPolicy != "" {
		sh.ReferrerPolicy = &cftypes.ResponseHeadersPolicyReferrerPolicy{
			ReferrerPolicy: cftypes.ReferrerPolicyList(cfg.ReferrerPolicy),
			Override:        aws.Bool(true),
		}
	}

	return policyCfg
}

// ensurePolicy looks up the existing "forge-security-headers" policy by name.
// If it exists, the policy is updated; otherwise a new one is created.
// Returns the policy ID.
func (c *AWSCloudFrontHeadersPolicyClient) ensurePolicy(
	ctx context.Context, policyCfg *cftypes.ResponseHeadersPolicyConfig,
) (string, error) {
	// Try to find existing policy by name.
	existingID, err := c.findPolicyByName(ctx, responseHeadersPolicyName)
	if err != nil {
		return "", fmt.Errorf("looking up policy: %w", err)
	}

	if existingID == "" {
		// Policy does not exist — create it.
		createOut, createErr := c.client.CreateResponseHeadersPolicy(ctx,
			&cloudfront.CreateResponseHeadersPolicyInput{
				ResponseHeadersPolicyConfig: policyCfg,
			})
		if createErr != nil {
			return "", fmt.Errorf("creating policy: %w", createErr)
		}
		if createOut.ResponseHeadersPolicy == nil || createOut.ResponseHeadersPolicy.Id == nil {
			return "", fmt.Errorf("creating policy: empty response from AWS")
		}
		return aws.ToString(createOut.ResponseHeadersPolicy.Id), nil
	}

	// Policy exists — get its ETag for conditional update.
	getOut, getErr := c.client.GetResponseHeadersPolicy(ctx,
		&cloudfront.GetResponseHeadersPolicyInput{
			Id: aws.String(existingID),
		})
	if getErr != nil {
		return "", fmt.Errorf("getting policy for update: %w", getErr)
	}

	_, updateErr := c.client.UpdateResponseHeadersPolicy(ctx,
		&cloudfront.UpdateResponseHeadersPolicyInput{
			Id:                          aws.String(existingID),
			ResponseHeadersPolicyConfig: policyCfg,
			IfMatch:                     getOut.ETag,
		})
	if updateErr != nil {
		return "", fmt.Errorf("updating policy: %w", updateErr)
	}

	return existingID, nil
}

// findPolicyByName iterates through custom response headers policies to find
// one matching the given name. Returns the policy ID or "" if not found.
func (c *AWSCloudFrontHeadersPolicyClient) findPolicyByName(ctx context.Context, name string) (string, error) {
	input := &cloudfront.ListResponseHeadersPoliciesInput{
		Type: cftypes.ResponseHeadersPolicyTypeCustom,
	}

	for {
		out, err := c.client.ListResponseHeadersPolicies(ctx, input)
		if err != nil {
			return "", fmt.Errorf("listing response headers policies: %w", err)
		}

		if out.ResponseHeadersPolicyList != nil {
			for _, item := range out.ResponseHeadersPolicyList.Items {
				if item.ResponseHeadersPolicy != nil &&
					item.ResponseHeadersPolicy.ResponseHeadersPolicyConfig != nil &&
					aws.ToString(item.ResponseHeadersPolicy.ResponseHeadersPolicyConfig.Name) == name {
					return aws.ToString(item.ResponseHeadersPolicy.Id), nil
				}
			}
		}

		// Check for pagination.
		if out.ResponseHeadersPolicyList == nil ||
			out.ResponseHeadersPolicyList.NextMarker == nil {
			break
		}
		input.Marker = out.ResponseHeadersPolicyList.NextMarker
	}

	return "", nil
}

// associatePolicyWithDistribution sets the ResponseHeadersPolicyId on the
// distribution's default cache behavior, if not already set to the given ID.
func (c *AWSCloudFrontHeadersPolicyClient) associatePolicyWithDistribution(
	ctx context.Context, distributionID, policyID string,
) error {
	// Get current distribution config.
	getOut, err := c.client.GetDistributionConfig(ctx, &cloudfront.GetDistributionConfigInput{
		Id: aws.String(distributionID),
	})
	if err != nil {
		return fmt.Errorf("getting distribution config: %w", err)
	}

	distConfig := getOut.DistributionConfig
	if distConfig == nil || distConfig.DefaultCacheBehavior == nil {
		return fmt.Errorf("distribution %q has no default cache behavior", distributionID)
	}
	defaultBehavior := distConfig.DefaultCacheBehavior

	// Check if already associated with this policy.
	if aws.ToString(defaultBehavior.ResponseHeadersPolicyId) == policyID {
		return nil
	}

	// Set the response headers policy ID.
	defaultBehavior.ResponseHeadersPolicyId = aws.String(policyID)

	// Update distribution.
	_, err = c.client.UpdateDistribution(ctx, &cloudfront.UpdateDistributionInput{
		Id:                 aws.String(distributionID),
		DistributionConfig: distConfig,
		IfMatch:            getOut.ETag,
	})
	if err != nil {
		return fmt.Errorf("updating distribution: %w", err)
	}

	return nil
}
