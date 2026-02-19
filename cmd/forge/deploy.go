package main

import (
	"context"
	"fmt"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	siteconfig "github.com/aellingwood/forge/internal/config"
	"github.com/aellingwood/forge/internal/deploy"
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
			Bucket:       cfg.Deploy.S3.Bucket,
			Region:       cfg.Deploy.S3.Region,
			Distribution: cfg.Deploy.CloudFront.DistributionID,
			URLRewrite:   cfg.Deploy.CloudFront.URLRewrite,
			DryRun:       dryRun,
			Verbose:      verbose,
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
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(deployCfg.Region),
		)
		if err != nil {
			return fmt.Errorf("loading AWS config: %w", err)
		}

		s3Client := deploy.NewAWSS3Client(s3.NewFromConfig(awsCfg), deployCfg.Bucket)

		var cfClient deploy.CloudFrontClient
		var cfFuncClient deploy.CloudFrontFunctionClient

		if deployCfg.Distribution != "" {
			cfSDK := cloudfront.NewFromConfig(awsCfg)
			cfClient = deploy.NewAWSCloudFrontClient(cfSDK)
			if deployCfg.URLRewrite {
				cfFuncClient = deploy.NewAWSCloudFrontFunctionClient(cfSDK)
			}
		}

		// 5. Deploy.
		fmt.Fprintf(cmd.OutOrStdout(), "Deploying to s3://%s ...\n", deployCfg.Bucket)
		result, err := deploy.Deploy(ctx, deployCfg, publicDir, s3Client, cfClient, cfFuncClient)
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
