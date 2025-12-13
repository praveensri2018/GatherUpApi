package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Client struct {
	S3        *s3.Client
	Bucket    string
	AccountID string
}

func NewR2Client(
	accountID, accessKey, secretKey, bucket string,
) (*R2Client, error) {

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				accessKey,
				secretKey,
				"",
			),
		),
		config.WithEndpointResolver(
			aws.EndpointResolverFunc(
				func(service, region string) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:               endpoint,
						HostnameImmutable: true,
					}, nil
				},
			),
		),
	)
	if err != nil {
		return nil, err
	}

	return &R2Client{
		S3:        s3.NewFromConfig(cfg),
		Bucket:    bucket,
		AccountID: accountID,
	}, nil
}

func (c *R2Client) Upload(
	ctx context.Context,
	key string,
	body io.Reader,
	size int64,
	contentType string,
) (string, error) {

	_, err := c.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.Bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}

	// Public R2 URL
	return fmt.Sprintf(
		"https://%s.r2.cloudflarestorage.com/%s/%s",
		c.AccountID,
		c.Bucket,
		key,
	), nil
}
