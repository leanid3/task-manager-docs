package minio

import (
	apperrors "app/internal/entity/errors"
	"app/pkg/logger"
	pkgminio "app/pkg/minio"
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type MinioAdapter struct {
	connector  *pkgminio.Connector
	client     *minio.Client
	bucketName string
	l          logger.Interface
}

func NewMinioAdapter(connector *pkgminio.Connector, l logger.Interface) *MinioAdapter {
	return &MinioAdapter{
		connector:  connector,
		client:     connector.Client(),
		bucketName: connector.BucketName(),
		l:          l,
	}
}

func (r *MinioAdapter) UploadStream(ctx context.Context, objectName string, reader io.Reader, objectSize int64, opts pkgminio.UploadOptions) (*pkgminio.ObjectInfo, error) {
	putOpts := minio.PutObjectOptions{
		ContentType:  opts.ContentType,
		UserMetadata: opts.Metadata,
		CacheControl: opts.CacheControl,
		StorageClass: opts.StorageClass,
		NumThreads:   4, //TODO сделать динамическим
	}

	//неизвесный размер файла
	if objectSize == -1 {
		r.l.Info("unknown size of the file", "objectName", objectName)
	}

	uploadInfo, err := r.client.PutObject(ctx, r.bucketName, objectName, reader, objectSize, putOpts)

	if err != nil {
		r.l.Error("failed to upload object", "error", err, "objectName", objectName)
		return nil, apperrors.Wrap(
			apperrors.CodeStorageError,
			"failed to upload object to storage",
			err,
		).WithDetails(map[string]interface{}{
			"objectName": objectName,
			"bucket":     r.bucketName,
		})
	}

	r.l.Info("object uploaded successfully",
		"objectName", objectName,
		"bucket", r.bucketName,
		"size", uploadInfo.Size,
		"etag", uploadInfo.ETag,
	)
	objectInfo := &pkgminio.ObjectInfo{
		Key:          objectName,
		Size:         uploadInfo.Size,
		ETag:         uploadInfo.ETag,
		ContentType:  opts.ContentType,
		LastModified: uploadInfo.LastModified,
		Metadata:     opts.Metadata,
	}
	return objectInfo, nil
}

func (r *MinioAdapter) DeleteObject(ctx context.Context, objectName string) error {
	err := r.client.RemoveObject(ctx, r.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		r.l.Error("failed to delete object", "error", err, "objectName", objectName)
		return apperrors.Wrap(
			apperrors.CodeStorageError,
			"failed to delete object",
			err,
		)
	}
	return nil
}

func (r *MinioAdapter) GetObjectMetadata(ctx context.Context, objectName string) (*pkgminio.ObjectInfo, error) {
	stat, err := r.client.StatObject(ctx, r.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		r.l.Error("failed to get object metadata", "error", err, "objectName", objectName)

		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return nil, apperrors.New(
				apperrors.CodeFileNotFound,
				"file not found in storage",
			).WithStatus(404).WithDetails(map[string]interface{}{
				"objectName": objectName,
			})
		}

		return nil, apperrors.Wrap(
			apperrors.CodeStorageError,
			"failed to get object metadata",
			err,
		)
	}

	return &pkgminio.ObjectInfo{
		Key:          objectName,
		Size:         stat.Size,
		ETag:         stat.ETag,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified,
		Metadata:     stat.UserMetadata,
	}, nil
}

func (r *MinioAdapter) ObjectExists(ctx context.Context, objectName string) (bool, error) {
	_, err := r.client.StatObject(ctx, r.bucketName, objectName, minio.StatObjectOptions{})

	if err != nil {
		r.l.Error("failed to check object existence", "error", err, "objectName", objectName)
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, apperrors.Wrap(
			apperrors.CodeStorageError,
			"failed to check object existence",
			err,
		)
	}
	return true, nil
}

// FileExists реализует интерфейс repository.Storage, проксируя вызов к ObjectExists.
func (r *MinioAdapter) FileExists(ctx context.Context, objectName string) (bool, error) {
	return r.ObjectExists(ctx, objectName)
}

func (r *MinioAdapter) ListObjects(ctx context.Context, prefix string, recursive bool) ([]pkgminio.ObjectInfo, error) {
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: recursive,
	}

	var objects []pkgminio.ObjectInfo

	objectCh := r.client.ListObjects(ctx, r.bucketName, opts)
	for object := range objectCh {
		if object.Err != nil {
			r.l.Error("failed to list objects", "error", object.Err, "prefix", prefix)
			return nil, apperrors.Wrap(
				apperrors.CodeStorageError,
				"failed to list objects",
				object.Err,
			)
		}
		objects = append(objects, pkgminio.ObjectInfo{
			Key:          object.Key,
			Size:         object.Size,
			ETag:         object.ETag,
			ContentType:  object.ContentType,
			LastModified: object.LastModified,
			Metadata:     object.UserMetadata,
		})
	}
	return objects, nil
}

// ListFiles реализует интерфейс repository.Storage, используя рекурсивный обход ListObjects.
func (r *MinioAdapter) ListFiles(ctx context.Context, prefix string) ([]pkgminio.ObjectInfo, error) {
	return r.ListObjects(ctx, prefix, true)
}

func (c *MinioAdapter) BucketName() string {
	return c.connector.BucketName()
}

func (r *MinioAdapter) GenerateStoragePath(taskID uuid.UUID, filename string) string {
	return fmt.Sprintf("%s/%s/%s",
		r.bucketName,
		taskID.String(),
		filename,
	)
}
