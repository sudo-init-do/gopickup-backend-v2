package storage

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"gopickup/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// R2Client wraps a minio S3 client configured for Cloudflare R2.
type R2Client struct {
	client *minio.Client
	bucket string
	base   string // public base URL, no trailing slash
}

// NewR2Client builds an R2-backed object storage client from config.
// Cloudflare R2's S3 endpoint is <account_id>.r2.cloudflarestorage.com.
func NewR2Client(cfg *config.Config) (*R2Client, error) {
	endpoint := fmt.Sprintf("%s.r2.cloudflarestorage.com", cfg.R2AccountID)
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.R2AccessKeyID, cfg.R2SecretAccessKey, ""),
		Secure: true,
		Region: "auto",
	})
	if err != nil {
		return nil, fmt.Errorf("r2: init client: %w", err)
	}
	return &R2Client{
		client: client,
		bucket: cfg.R2Bucket,
		base:   strings.TrimSuffix(cfg.R2PublicBaseURL, "/"),
	}, nil
}

// Upload stores data under key and returns its public URL.
func (r *R2Client) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := r.client.PutObject(ctx, r.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType:  contentType,
		CacheControl: "public, max-age=31536000, immutable",
	})
	if err != nil {
		return "", fmt.Errorf("r2: put object %q: %w", key, err)
	}
	return fmt.Sprintf("%s/%s", r.base, key), nil
}
