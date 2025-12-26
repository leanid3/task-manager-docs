package minio

import (
	"app/pkg/logger"
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// TODO сделать обработку ошибок
type Connector struct {
	client *minio.Client
	cfg    *Config
	l      logger.Interface
}

func NewConnector(cfg *Config, l logger.Interface) (*Connector, error) {

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		l.Error("failed to create minio client", "error", err)
		return nil, err
	}
	connector := &Connector{
		client: client,
		cfg:    cfg,
		l:      l,
	}

	if err := connector.ensureBucket(context.Background()); err != nil {
		l.Error("failed to ensure bucket exists", "error", err, "bucket", cfg.Bucket)
		return nil, err
	}

	return connector, nil
}

func (c *Connector) ensureBucket(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	exists, err := c.client.BucketExists(ctx, c.cfg.Bucket)
	if err != nil {
		return fmt.Errorf("ошибка при проверке существования бакета: %w", err)
	}

	if !exists {
		err = c.client.MakeBucket(ctx, c.cfg.Bucket, minio.MakeBucketOptions{
			Region: c.cfg.Region,
		})
		if err != nil {
			return fmt.Errorf("ошибка при создании бакета: %w", err)
		}
	}
	return nil
}

func (c *Connector) Client() *minio.Client {
	return c.client
}

func (c *Connector) BucketName() string {
	return c.cfg.Bucket
}

func (c *Connector) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.client.BucketExists(ctx, c.cfg.Bucket)
	if err != nil {
		return fmt.Errorf("minio не отвечает %w", err)
	}

	return nil
}
