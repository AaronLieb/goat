package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func LoadConfig(ctx context.Context, debug bool) (aws.Config, error) {
	var logMode aws.ClientLogMode
	if debug {
		logMode = aws.LogRequestWithBody | aws.LogResponseWithBody
	}
	cfg, err := config.LoadDefaultConfig(ctx, config.WithClientLogMode(logMode))
	return cfg, err
}

func AccessKey(ctx context.Context, cfg aws.Config) (string, error) {
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return "", err
	}
	return creds.AccessKeyID, nil
}
