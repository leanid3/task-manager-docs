package repository

import (
	"app/pkg/minio"
	"context"
	"io"

	"github.com/google/uuid"
)

type Storage interface {
	UploadStream(ctx context.Context, objectName string, reader io.Reader, objectSize int64, opts minio.UploadOptions) (*minio.ObjectInfo, error)
	DeleteObject(ctx context.Context, objectName string) error
	GetObjectMetadata(ctx context.Context, objectName string) (*minio.ObjectInfo, error)
	FileExists(ctx context.Context, objectName string) (bool, error)
	ListFiles(ctx context.Context, prefix string) ([]minio.ObjectInfo, error)
	BucketName() string
	GenerateStoragePath(taskID uuid.UUID, filename string) string
}
