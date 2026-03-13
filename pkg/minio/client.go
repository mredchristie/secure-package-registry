// Package minio provides a thin wrapper around the MinIO Go SDK for
// storing and retrieving behavioral analysis artifacts.
package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds MinIO connection parameters.
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
}

// Client wraps a MinIO client and a default bucket.
type Client struct {
	mc     *minio.Client
	bucket string
}

// NewClient creates a MinIO client and ensures the configured bucket exists.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating minio client: %w", err)
	}

	c := &Client{mc: mc, bucket: cfg.Bucket}
	if err := c.ensureBucket(ctx); err != nil {
		return nil, err
	}

	return c, nil
}

// Bucket returns the configured bucket name.
func (c *Client) Bucket() string {
	return c.bucket
}

// PutObject uploads data to the configured bucket under the given key.
func (c *Client) PutObject(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := c.mc.PutObject(ctx, c.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("uploading object %s/%s: %w", c.bucket, key, err)
	}
	return nil
}

// GetObject downloads an object from the configured bucket.
func (c *Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting object %s/%s: %w", c.bucket, key, err)
	}
	defer func() { _ = obj.Close() }()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("reading object %s/%s: %w", c.bucket, key, err)
	}
	return data, nil
}

// ensureBucket creates the bucket unconditionally and ignores
// "already exists" errors to avoid a TOCTOU race between
// BucketExists and MakeBucket.
func (c *Client) ensureBucket(ctx context.Context) error {
	err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{})
	if err != nil {
		// Check if the bucket already exists — not an error.
		if errResp := minio.ToErrorResponse(err); errResp.Code == "BucketAlreadyOwnedByYou" || errResp.Code == "BucketAlreadyExists" {
			return nil
		}
		return fmt.Errorf("creating bucket %s: %w", c.bucket, err)
	}
	return nil
}
