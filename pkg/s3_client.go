package blobcache

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	s3Client *s3.Client
	source   struct {
		BucketName  string
		Path        string
		Region      string
		EndpointURL string
		AccessKey   string
		SecretKey   string
	}
}

func NewS3Client(ctx context.Context, source struct {
	BucketName  string
	Path        string
	Region      string
	EndpointURL string
	AccessKey   string
	SecretKey   string
}) (*S3Client, error) {
	Logger.Infof("NewS3Client[ACK] - [%v]", source)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(source.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			source.AccessKey,
			source.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// If a custom endpoint is provided, use it
	if source.EndpointURL != "" {
		cfg.BaseEndpoint = aws.String(source.EndpointURL)
	}

	s3Client := s3.NewFromConfig(cfg)

	return &S3Client{
		s3Client: s3Client,
	}, nil
}

func (c *S3Client) S3Client() *s3.Client {
	return c.s3Client
}

func (c *S3Client) BucketName() string {
	return c.source.BucketName
}

func (c *S3Client) Head(ctx context.Context, key string) (bool, *s3.HeadObjectOutput, error) {
	output, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.source.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, nil, err
	}

	return true, output, nil
}

func (c *S3Client) DownloadIntoBuffer(ctx context.Context, key string, buffer *bytes.Buffer) error {
	resp, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.source.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	_, err = io.Copy(buffer, resp.Body)
	return err
}
