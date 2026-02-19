package deploy

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// s3API is the subset of the S3 SDK client used by AWSS3Client.
type s3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// AWSS3Client implements S3Client using the AWS SDK v2.
type AWSS3Client struct {
	client s3API
	bucket string
}

// NewAWSS3Client creates a new AWSS3Client.
func NewAWSS3Client(client s3API, bucket string) *AWSS3Client {
	return &AWSS3Client{client: client, bucket: bucket}
}

// PutObject uploads an object to S3 with the given key, content type,
// cache control, and SHA-256 hash stored as metadata.
func (c *AWSS3Client) PutObject(ctx context.Context, key string, body io.Reader, contentType, cacheControl, sha256Hash string) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(c.bucket),
		Key:          aws.String(key),
		Body:         body,
		ContentType:  aws.String(contentType),
		CacheControl: aws.String(cacheControl),
		Metadata: map[string]string{
			"sha256": sha256Hash,
		},
	})
	if err != nil {
		return fmt.Errorf("s3 PutObject %q: %w", key, err)
	}
	return nil
}

// DeleteObject deletes an object from S3.
func (c *AWSS3Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3 DeleteObject %q: %w", key, err)
	}
	return nil
}

// ListObjects lists all objects in the bucket and returns a map of
// key -> SHA-256 hash (from object metadata).
func (c *AWSS3Client) ListObjects(ctx context.Context, prefix string) (map[string]string, error) {
	result := make(map[string]string)

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	for {
		out, err := c.client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("s3 ListObjectsV2: %w", err)
		}

		for _, obj := range out.Contents {
			key := aws.ToString(obj.Key)
			// Use ETag as a change-detection hash (stripped of quotes).
			etag := aws.ToString(obj.ETag)
			if len(etag) >= 2 && etag[0] == '"' && etag[len(etag)-1] == '"' {
				etag = etag[1 : len(etag)-1]
			}
			result[key] = etag
		}

		if !aws.ToBool(out.IsTruncated) {
			break
		}
		input.ContinuationToken = out.NextContinuationToken
	}

	return result, nil
}

// cfInvalidationAPI is the subset of the CloudFront SDK client used by AWSCloudFrontClient.
type cfInvalidationAPI interface {
	CreateInvalidation(ctx context.Context, params *cloudfront.CreateInvalidationInput, optFns ...func(*cloudfront.Options)) (*cloudfront.CreateInvalidationOutput, error)
}

// AWSCloudFrontClient implements CloudFrontClient using the AWS SDK v2.
type AWSCloudFrontClient struct {
	client cfInvalidationAPI
}

// NewAWSCloudFrontClient creates a new AWSCloudFrontClient.
func NewAWSCloudFrontClient(client cfInvalidationAPI) *AWSCloudFrontClient {
	return &AWSCloudFrontClient{client: client}
}

// CreateInvalidation creates a CloudFront invalidation for the given paths.
func (c *AWSCloudFrontClient) CreateInvalidation(ctx context.Context, distributionID string, paths []string) error {
	qty := int32(len(paths))
	callerRef := fmt.Sprintf("forge-%d", time.Now().UnixNano())

	_, err := c.client.CreateInvalidation(ctx, &cloudfront.CreateInvalidationInput{
		DistributionId: aws.String(distributionID),
		InvalidationBatch: &cftypes.InvalidationBatch{
			CallerReference: aws.String(callerRef),
			Paths: &cftypes.Paths{
				Quantity: &qty,
				Items:    paths,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("cloudfront CreateInvalidation: %w", err)
	}
	return nil
}
