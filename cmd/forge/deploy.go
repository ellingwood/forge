package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	siteconfig "github.com/aellingwood/forge/internal/config"
	"github.com/aellingwood/forge/internal/deploy"
	"github.com/aellingwood/forge/internal/security"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the site",
	Long:  "Deploy the built site to the configured destination.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Load site config.
		configPath, _ := cmd.Root().PersistentFlags().GetString("config")
		cfg, err := siteconfig.Load(configPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		deployCfg := deploy.DeployConfig{
			Bucket:          cfg.Deploy.S3.Bucket,
			Region:          cfg.Deploy.S3.Region,
			Endpoint:        cfg.Deploy.Endpoint,
			Profile:         cfg.Deploy.Profile,
			Distribution:    cfg.Deploy.CloudFront.DistributionID,
			URLRewrite:      cfg.Deploy.CloudFront.URLRewrite,
			SecurityHeaders: cfg.Deploy.CloudFront.SecurityHeaders,
			DryRun:          dryRun,
			Verbose:         verbose,
		}

		// Build security headers config if enabled.
		if deployCfg.SecurityHeaders {
			cspPolicy := security.ProdPolicy(&cfg.Security.CSP)
			deployCfg.SecurityHeadersCfg = deploy.ResponseHeadersConfig{
				CSP:                 cspPolicy.String(),
				HSTSMaxAge:          cfg.Security.HSTS.MaxAge,
				HSTSSubDomains:      cfg.Security.HSTS.IncludeSubDomains,
				HSTSPreload:         cfg.Security.HSTS.Preload,
				XContentTypeNosniff: true,
				XFrameOptions:       "DENY",
				ReferrerPolicy:      "strict-origin-when-cross-origin",
			}
		}

		if deployCfg.Bucket == "" {
			return fmt.Errorf("deploy.s3.bucket is required in config")
		}
		if deployCfg.Region == "" {
			return fmt.Errorf("deploy.s3.region is required in config")
		}

		// 2. Determine output directory.
		publicDir := "public"

		// 3. Check that public dir exists.
		if _, err := os.Stat(publicDir); os.IsNotExist(err) {
			return fmt.Errorf("output directory %q not found; run 'forge build' first", publicDir)
		}

		// 4. Build AWS SDK clients.
		ctx := context.Background()
		awsOpts := []func(*awsconfig.LoadOptions) error{
			awsconfig.WithRegion(deployCfg.Region),
		}
		if deployCfg.Profile != "" {
			awsOpts = append(awsOpts, awsconfig.WithSharedConfigProfile(deployCfg.Profile))
		}
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsOpts...)
		if err != nil {
			return fmt.Errorf("loading AWS config: %w", err)
		}

		s3Opts := []func(*s3.Options){}
		if deployCfg.Endpoint != "" {
			s3Opts = append(s3Opts, func(o *s3.Options) {
				o.BaseEndpoint = aws.String(deployCfg.Endpoint)
				o.UsePathStyle = true
			})
		}
		s3Client := deploy.NewAWSS3Client(s3.NewFromConfig(awsCfg, s3Opts...), deployCfg.Bucket)

		var cfClient deploy.CloudFrontClient
		var cfFuncClient deploy.CloudFrontFunctionClient
		var cfHeadersClient deploy.CloudFrontHeadersPolicyClient

		if deployCfg.Distribution != "" {
			cfOpts := []func(*cloudfront.Options){}
			if deployCfg.Endpoint != "" {
				cfOpts = append(cfOpts, func(o *cloudfront.Options) {
					o.BaseEndpoint = aws.String(deployCfg.Endpoint)
				})
			}
			cfSDK := cloudfront.NewFromConfig(awsCfg, cfOpts...)
			cfClient = deploy.NewAWSCloudFrontClient(cfSDK)
			if deployCfg.URLRewrite {
				cfFuncClient = deploy.NewAWSCloudFrontFunctionClient(cfSDK)
			}
			if deployCfg.SecurityHeaders {
				cfHeadersClient = deploy.NewAWSCloudFrontHeadersPolicyClient(cfSDK)
			}
		}

		// 5. Deploy.
		fmt.Fprintf(cmd.OutOrStdout(), "Deploying to s3://%s ...\n", deployCfg.Bucket)
		result, err := deploy.Deploy(ctx, deployCfg, publicDir, s3Client, cfClient, cfFuncClient, cfHeadersClient)
		if err != nil {
			return fmt.Errorf("deploy failed: %w", err)
		}

		// 6. Print result summary.
		fmt.Fprintf(cmd.OutOrStdout(), "Deploy complete: %d uploaded, %d deleted, %d skipped\n",
			result.Uploaded, result.Deleted, result.Skipped)

		for _, e := range result.Errors {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", e)
		}

		return nil
	},
}

func init() {
	deployCmd.Flags().Bool("dry-run", false, "show what would be deployed without deploying")

	rootCmd.AddCommand(deployCmd)
}
