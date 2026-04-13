package service

import (
	"app/pkg/minio"
	"context"
	"io"

	"github.com/google/uuid"
)

// Storage — абстракция над файловым хранилищем (MinIO/S3).
// UseCase загружает файлы через этот интерфейс, не зная о MinIO.
type Storage interface {
	// UploadStream загружает поток данных в хранилище
	UploadStream(ctx context.Context, objectName string, reader io.Reader, objectSize int64, opts minio.UploadOptions) (*minio.ObjectInfo, error)
	// DeleteObject удаляет объект из хранилища
	DeleteObject(ctx context.Context, objectName string) error
	// GetObjectMetadata получает метаданные объекта
	GetObjectMetadata(ctx context.Context, objectName string) (*minio.ObjectInfo, error)
	// FileExists проверяет существование файла
	FileExists(ctx context.Context, objectName string) (bool, error)
	// ListFiles возвращает список файлов по префиксу
	ListFiles(ctx context.Context, prefix string) ([]minio.ObjectInfo, error)
	// BucketName возвращает имя бакета
	BucketName() string
	// GenerateStoragePath генерирует путь для хранения
	GenerateStoragePath(taskID uuid.UUID, filename string) string
}
