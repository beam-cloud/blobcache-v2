package blobcache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	Client              *s3.Client
	Source              S3SourceConfig
	DownloadConcurrency int64
	DownloadChunkSize   int64
}

type S3SourceConfig struct {
	BucketName  string
	Region      string
	EndpointURL string
	AccessKey   string
	SecretKey   string
}

func NewS3Client(ctx context.Context, sourceConfig S3SourceConfig, serverConfig BlobCacheServerConfig) (*S3Client, error) {
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
		Client:              s3Client,
		Source:              sourceConfig,
		DownloadConcurrency: serverConfig.S3DownloadConcurrency,
		DownloadChunkSize:   serverConfig.S3DownloadChunkSize,
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
	if size <= 0 {
		return fmt.Errorf("invalid object size: %d", size)
	}

	numChunks := int((size + c.DownloadChunkSize - 1) / c.DownloadChunkSize)
	chunks := make([][]byte, numChunks)
	sem := make(chan struct{}, c.DownloadConcurrency)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)

	for i := 0; i < numChunks; i++ {
		wg.Add(1)

		go func(i int) {
			sem <- struct{}{}
			defer func() { <-sem }()
			defer wg.Done()

			if ctx.Err() != nil {
				return
			}

			start := int64(i) * c.DownloadChunkSize
			end := start + c.DownloadChunkSize - 1
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
				select {
				case errCh <- fmt.Errorf("range request failed for %s: %w", rangeHeader, err):
					cancel()
				default:
				}
				return
			}
			defer resp.Body.Close()

			part := make([]byte, end-start+1)
			n, err := io.ReadFull(resp.Body, part)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				select {
				case errCh <- fmt.Errorf("error reading range %s: %w", rangeHeader, err):
					cancel()
				default:
				}
				return
			}

			chunks[i] = part[:n]
		}(i)
	}

	wg.Wait()
	close(errCh)

	if err, ok := <-errCh; ok {
		return err
	}

	buffer.Reset()
	for _, chunk := range chunks {
		buffer.Write(chunk)
	}

	return nil
}
