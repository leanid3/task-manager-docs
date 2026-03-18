package minio

import (
	"app/pkg/logger"
	"app/pkg/metrics"
	pkgminio "app/pkg/minio"
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

// MinioAdapterWithMetrics - декоратор для MinioAdapter с метриками
type MinioAdapterWithMetrics struct {
	adapter *MinioAdapter
}

func NewMinioAdapterWithMetrics(connector *pkgminio.Connector, l logger.Interface) *MinioAdapterWithMetrics {
	return &MinioAdapterWithMetrics{
		adapter: &MinioAdapter{
			connector:  connector,
			client:     connector.Client(),
			bucketName: connector.BucketName(),
			l:          l,
		},
	}
}

func (r *MinioAdapterWithMetrics) UploadStream(ctx context.Context, objectName string, reader io.Reader, objectSize int64, opts pkgminio.UploadOptions) (*pkgminio.ObjectInfo, error) {
	start := time.Now()
	result, err := r.adapter.UploadStream(ctx, objectName, reader, objectSize, opts)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveMinIOOperationDuration("upload", r.adapter.bucketName, duration)
		metrics.IncTasksTotal("minio_upload", "error")
	} else {
		metrics.ObserveMinIOOperationDuration("upload", r.adapter.bucketName, duration)
		metrics.IncMinIOBytesTransferred("upload", r.adapter.bucketName, float64(objectSize))
		metrics.IncTasksTotal("minio_upload", "success")
	}

	return result, err
}

func (r *MinioAdapterWithMetrics) DeleteObject(ctx context.Context, objectName string) error {
	start := time.Now()
	err := r.adapter.DeleteObject(ctx, objectName)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveMinIOOperationDuration("delete", r.adapter.bucketName, duration)
		metrics.IncTasksTotal("minio_delete", "error")
	} else {
		metrics.ObserveMinIOOperationDuration("delete", r.adapter.bucketName, duration)
		metrics.IncTasksTotal("minio_delete", "success")
	}

	return err
}

func (r *MinioAdapterWithMetrics) GetObjectMetadata(ctx context.Context, objectName string) (*pkgminio.ObjectInfo, error) {
	start := time.Now()
	result, err := r.adapter.GetObjectMetadata(ctx, objectName)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveMinIOOperationDuration("stat", r.adapter.bucketName, duration)
		metrics.IncTasksTotal("minio_stat", "error")
	} else {
		metrics.ObserveMinIOOperationDuration("stat", r.adapter.bucketName, duration)
		metrics.IncTasksTotal("minio_stat", "success")
	}

	return result, err
}

func (r *MinioAdapterWithMetrics) ObjectExists(ctx context.Context, objectName string) (bool, error) {
	start := time.Now()
	result, err := r.adapter.ObjectExists(ctx, objectName)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveMinIOOperationDuration("exists", r.adapter.bucketName, duration)
		metrics.IncTasksTotal("minio_exists", "error")
	} else {
		metrics.ObserveMinIOOperationDuration("exists", r.adapter.bucketName, duration)
		metrics.IncTasksTotal("minio_exists", "success")
	}

	return result, err
}

func (r *MinioAdapterWithMetrics) FileExists(ctx context.Context, objectName string) (bool, error) {
	return r.ObjectExists(ctx, objectName)
}

func (r *MinioAdapterWithMetrics) ListObjects(ctx context.Context, prefix string, recursive bool) ([]pkgminio.ObjectInfo, error) {
	start := time.Now()
	result, err := r.adapter.ListObjects(ctx, prefix, recursive)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveMinIOOperationDuration("list", r.adapter.bucketName, duration)
		metrics.IncTasksTotal("minio_list", "error")
	} else {
		metrics.ObserveMinIOOperationDuration("list", r.adapter.bucketName, duration)
		metrics.IncTasksTotal("minio_list", "success")
	}

	return result, err
}

func (r *MinioAdapterWithMetrics) ListFiles(ctx context.Context, prefix string) ([]pkgminio.ObjectInfo, error) {
	return r.ListObjects(ctx, prefix, true)
}

func (r *MinioAdapterWithMetrics) BucketName() string {
	return r.adapter.BucketName()
}

func (r *MinioAdapterWithMetrics) GenerateStoragePath(taskID uuid.UUID, filename string) string {
	return r.adapter.GenerateStoragePath(taskID, filename)
}
