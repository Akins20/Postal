// Package storage abstracts S3-compatible object storage for the media pipeline:
// uploading, streaming, and deleting binary objects by key. Production uses
// Cloudflare R2 (S3 API; endpoint <account>.r2.cloudflarestorage.com, region
// "auto", TLS); local dev uses the docker-compose MinIO as an R2-compatible
// stand-in. Both speak the S3 API, so one client serves both.
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Storage stores and retrieves binary objects by key.
type Storage interface {
	// Put stores size bytes from r under key with the given content type.
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	// Get opens the object at key for reading; the caller closes it.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the object at key (no error if absent).
	Delete(ctx context.Context, key string) error
}

// Config configures the S3-compatible client (R2 or MinIO).
type Config struct {
	Endpoint  string // R2: <account>.r2.cloudflarestorage.com; MinIO: host:port
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string // R2: "auto"; MinIO: empty/"us-east-1"
	UseSSL    bool   // R2: true; local MinIO: false
}

// Client is an S3-compatible Storage (Cloudflare R2 in prod, MinIO in dev).
type Client struct {
	client *minio.Client
	bucket string
}

// New builds an S3-compatible Storage client and ensures the bucket exists.
func New(ctx context.Context, cfg Config) (*Client, error) {
	c, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("creating storage client: %w", err)
	}
	exists, err := c.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("checking bucket: %w", err)
	}
	if !exists {
		if err := c.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("creating bucket %q: %w", cfg.Bucket, err)
		}
	}
	return &Client{client: c, bucket: cfg.Bucket}, nil
}

// Put uploads an object.
func (c *Client) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	if _, err := c.client.PutObject(ctx, c.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return fmt.Errorf("storage put %q: %w", key, err)
	}
	return nil
}

// Get opens an object for reading.
func (c *Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage get %q: %w", key, err)
	}
	return obj, nil
}

// Delete removes an object.
func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.client.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("storage delete %q: %w", key, err)
	}
	return nil
}
