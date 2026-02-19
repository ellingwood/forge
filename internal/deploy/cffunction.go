package deploy

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
)

// cfAPI is the subset of the CloudFront SDK client used by AWSCloudFrontFunctionClient.
type cfAPI interface {
	DescribeFunction(ctx context.Context, params *cloudfront.DescribeFunctionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.DescribeFunctionOutput, error)
	CreateFunction(ctx context.Context, params *cloudfront.CreateFunctionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.CreateFunctionOutput, error)
	UpdateFunction(ctx context.Context, params *cloudfront.UpdateFunctionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.UpdateFunctionOutput, error)
	PublishFunction(ctx context.Context, params *cloudfront.PublishFunctionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.PublishFunctionOutput, error)
	GetDistributionConfig(ctx context.Context, params *cloudfront.GetDistributionConfigInput, optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionConfigOutput, error)
	UpdateDistribution(ctx context.Context, params *cloudfront.UpdateDistributionInput, optFns ...func(*cloudfront.Options)) (*cloudfront.UpdateDistributionOutput, error)
}

// AWSCloudFrontFunctionClient implements CloudFrontFunctionClient using the AWS SDK v2.
type AWSCloudFrontFunctionClient struct {
	client cfAPI
}

// NewAWSCloudFrontFunctionClient creates a new AWSCloudFrontFunctionClient.
func NewAWSCloudFrontFunctionClient(client cfAPI) *AWSCloudFrontFunctionClient {
	return &AWSCloudFrontFunctionClient{client: client}
}

// EnsureURLRewriteFunction creates or updates a CloudFront Function, publishes
// it to LIVE, and associates it with the distribution's default cache behavior
// as a viewer-request function. Returns the function ARN.
//
// All mutations are idempotent: re-running with the same inputs is safe.
func (c *AWSCloudFrontFunctionClient) EnsureURLRewriteFunction(
	ctx context.Context, distributionID, functionName, functionCode string,
) (string, error) {
	// Step 1: Create or update the function and get its ETag.
	functionARN, etag, err := c.ensureFunction(ctx, functionName, functionCode)
	if err != nil {
		return "", fmt.Errorf("ensuring function %q: %w", functionName, err)
	}

	// Step 2: Publish from DEVELOPMENT to LIVE.
	if err := c.publishFunction(ctx, functionName, etag); err != nil {
		return "", fmt.Errorf("publishing function %q: %w", functionName, err)
	}

	// Step 3: Associate with the distribution's default cache behavior.
	if err := c.associateWithDistribution(ctx, distributionID, functionARN); err != nil {
		return "", fmt.Errorf("associating function with distribution %q: %w", distributionID, err)
	}

	return functionARN, nil
}

// ensureFunction checks if the function exists; creates it if not, updates it
// if so. Returns the function ARN and the current ETag.
func (c *AWSCloudFrontFunctionClient) ensureFunction(
	ctx context.Context, name, code string,
) (arn string, etag string, err error) {
	funcConfig := &cftypes.FunctionConfig{
		Comment: aws.String("Forge URL rewrite: appends index.html for clean URLs"),
		Runtime: cftypes.FunctionRuntimeCloudfrontJs20,
	}

	// Try to describe existing function.
	descOut, descErr := c.client.DescribeFunction(ctx, &cloudfront.DescribeFunctionInput{
		Name:  aws.String(name),
		Stage: cftypes.FunctionStageDevelopment,
	})

	var notFound *cftypes.NoSuchFunctionExists
	if descErr != nil && !errors.As(descErr, &notFound) {
		return "", "", fmt.Errorf("describing function: %w", descErr)
	}

	if errors.As(descErr, &notFound) {
		// Function doesn't exist — create it.
		createOut, createErr := c.client.CreateFunction(ctx, &cloudfront.CreateFunctionInput{
			Name:           aws.String(name),
			FunctionCode:   []byte(code),
			FunctionConfig: funcConfig,
		})
		if createErr != nil {
			return "", "", fmt.Errorf("creating function: %w", createErr)
		}
		return aws.ToString(createOut.FunctionSummary.FunctionMetadata.FunctionARN),
			aws.ToString(createOut.ETag), nil
	}

	// Function exists — update it.
	updateOut, updateErr := c.client.UpdateFunction(ctx, &cloudfront.UpdateFunctionInput{
		Name:           aws.String(name),
		FunctionCode:   []byte(code),
		FunctionConfig: funcConfig,
		IfMatch:        descOut.ETag,
	})
	if updateErr != nil {
		return "", "", fmt.Errorf("updating function: %w", updateErr)
	}
	return aws.ToString(updateOut.FunctionSummary.FunctionMetadata.FunctionARN),
		aws.ToString(updateOut.ETag), nil
}

// publishFunction publishes the function from DEVELOPMENT to LIVE.
func (c *AWSCloudFrontFunctionClient) publishFunction(ctx context.Context, name, etag string) error {
	_, err := c.client.PublishFunction(ctx, &cloudfront.PublishFunctionInput{
		Name:    aws.String(name),
		IfMatch: aws.String(etag),
	})
	if err != nil {
		return fmt.Errorf("publishing: %w", err)
	}
	return nil
}

// associateWithDistribution attaches the function to the distribution's default
// cache behavior as a viewer-request function, if not already associated.
func (c *AWSCloudFrontFunctionClient) associateWithDistribution(
	ctx context.Context, distributionID, functionARN string,
) error {
	// Get current distribution config.
	getOut, err := c.client.GetDistributionConfig(ctx, &cloudfront.GetDistributionConfigInput{
		Id: aws.String(distributionID),
	})
	if err != nil {
		return fmt.Errorf("getting distribution config: %w", err)
	}

	distConfig := getOut.DistributionConfig
	defaultBehavior := distConfig.DefaultCacheBehavior

	// Check if already associated.
	if defaultBehavior.FunctionAssociations != nil {
		for _, assoc := range defaultBehavior.FunctionAssociations.Items {
			if assoc.EventType == cftypes.EventTypeViewerRequest &&
				aws.ToString(assoc.FunctionARN) == functionARN {
				// Already associated — nothing to do.
				return nil
			}
		}
	}

	// Build the new association.
	newAssoc := cftypes.FunctionAssociation{
		EventType:   cftypes.EventTypeViewerRequest,
		FunctionARN: aws.String(functionARN),
	}

	// Remove any existing viewer-request function association (replace, don't stack).
	var kept []cftypes.FunctionAssociation
	if defaultBehavior.FunctionAssociations != nil {
		for _, assoc := range defaultBehavior.FunctionAssociations.Items {
			if assoc.EventType != cftypes.EventTypeViewerRequest {
				kept = append(kept, assoc)
			}
		}
	}
	kept = append(kept, newAssoc)

	qty := int32(len(kept))
	defaultBehavior.FunctionAssociations = &cftypes.FunctionAssociations{
		Quantity: &qty,
		Items:    kept,
	}

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
