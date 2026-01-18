package config

import (
	"context"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	EnvCFAccessKey = "CLOUDFLARE_ACCESS_KEY"
	EnvCFSecretKey = "CLOUDFLARE_SECRET_KEY"
	EnvCFEndpoint  = "CLOUDFLARE_ENDPOINT"
	EnvCFBucket    = "CLOUDFLARE_BUCKET_NAME"
	EnvBucketUrl   = "BUCKET_URL"
)

var (
	BucketURL string
)

type (
	Client interface {
		Upload(ctx context.Context, key, mime string, body io.Reader) error
		Download(ctx context.Context, key string) ([]byte, error)
		Delete(ctx context.Context, key string) error
	}
	storageClient struct {
		client *s3.Client
		bucket string
	}
	StorageConfig struct {
		CloudflareAccessKey string
		CloudflareSecretKey string
		CloudflareEndpoint  string
		CloudflareBucket    string
	}
)

func NewStorageClient(client *s3.Client, bucket string) *storageClient {
	return &storageClient{
		client: client,
		bucket: bucket,
	}
}

func LoadStorageConfig() (*StorageConfig, error) {
	BucketURL = getEnv(EnvBucketUrl, "")
	c := &StorageConfig{
		CloudflareAccessKey: getEnv(EnvCFAccessKey, ""),
		CloudflareSecretKey: getEnv(EnvCFSecretKey, ""),
		CloudflareEndpoint:  "https://" + getEnv(EnvCFEndpoint, ""),
		CloudflareBucket:    getEnv(EnvCFBucket, ""),
	}
	if c.CloudflareAccessKey == "" || c.CloudflareSecretKey == "" || c.CloudflareEndpoint == "" || c.CloudflareBucket == "" {
		return nil, errors.New("missing required cloudflare storage env vars")
	}
	return c, nil
}

func InitStorageClient(ctx context.Context, accessKey, secretKey, endpoint string) (*s3.Client, error) {

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, err
	}
	s := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return s, nil
}

func (s *storageClient) Upload(ctx context.Context, key, mime string, body io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(mime),
	})
	return err
}

func (s *storageClient) Download(ctx context.Context, key string) ([]byte, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

func (s *storageClient) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}
