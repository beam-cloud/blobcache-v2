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

const (
	downloadChunkSize = 64 * 1024 * 1024 // 64MB
)

type S3Client struct {
	Client *s3.Client
	Source S3SourceConfig
}

type S3SourceConfig struct {
	BucketName  string
	Region      string
	EndpointURL string
	AccessKey   string
	SecretKey   string
}

func NewS3Client(ctx context.Context, sourceConfig S3SourceConfig) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(sourceConfig.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			sourceConfig.AccessKey,
			sourceConfig.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	if sourceConfig.EndpointURL != "" {
		cfg.BaseEndpoint = aws.String(sourceConfig.EndpointURL)
	}

	s3Client := s3.NewFromConfig(cfg)
	return &S3Client{
		Client: s3Client,
		Source: sourceConfig,
	}, nil
}

func (c *S3Client) GetClient() *s3.Client {
	return c.Client
}

func (c *S3Client) BucketName() string {
	return c.Source.BucketName
}

func (c *S3Client) Head(ctx context.Context, key string) (bool, *s3.HeadObjectOutput, error) {
	output, err := c.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.Source.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return false, nil, err
	}

	return true, output, nil
}

func (c *S3Client) DownloadIntoBuffer(ctx context.Context, key string, buffer *bytes.Buffer) error {
	ok, head, err := c.Head(ctx, key)
	if err != nil || !ok {
		return err
	}
	size := aws.ToInt64(head.ContentLength)

	buffer.Reset()
	var start int64 = 0

	for start < size {
		end := start + downloadChunkSize - 1
		if end >= size {
			end = size - 1
		}

		rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
		resp, err := c.Client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(c.Source.BucketName),
			Key:    aws.String(key),
			Range:  &rangeHeader,
		})
		if err != nil {
			return err
		}

		n, err := io.Copy(buffer, resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		start += n
	}

	return nil
}
