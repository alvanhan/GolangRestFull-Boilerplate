package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"file-management-service/config"
)

type MinioStorage struct {
	client     *minio.Client
	bucketName string
	cfg        *config.MinIOConfig
}

type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

func NewMinioStorage(cfg *config.MinIOConfig) (*MinioStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("creating minio client: %w", err)
	}

	s := &MinioStorage{
		client:     client,
		bucketName: cfg.BucketName,
		cfg:        cfg,
	}

	if err := s.EnsureBucket(context.Background()); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *MinioStorage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("checking bucket existence: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{
			Region: s.cfg.Region,
		}); err != nil {
			return fmt.Errorf("creating bucket %q: %w", s.bucketName, err)
		}
	}
	return nil
}

func (s *MinioStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucketName, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("uploading object %q: %w", key, err)
	}
	return nil
}

func (s *MinioStorage) Download(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	obj, err := s.client.GetObject(ctx, s.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("getting object %q: %w", key, err)
	}

	info, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, 0, fmt.Errorf("stat object %q: %w", key, err)
	}

	return obj, info.Size, nil
}

func (s *MinioStorage) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucketName, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("deleting object %q: %w", key, err)
	}
	return nil
}

func (s *MinioStorage) DeleteMultiple(ctx context.Context, keys []string) error {
	objectsCh := make(chan minio.ObjectInfo)

	go func() {
		defer close(objectsCh)
		for _, key := range keys {
			objectsCh <- minio.ObjectInfo{Key: key}
		}
	}()

	for removeErr := range s.client.RemoveObjects(ctx, s.bucketName, objectsCh, minio.RemoveObjectsOptions{}) {
		if removeErr.Err != nil {
			return fmt.Errorf("deleting object %q: %w", removeErr.ObjectName, removeErr.Err)
		}
	}
	return nil
}

func (s *MinioStorage) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucketName, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("generating presigned URL for %q: %w", key, err)
	}
	return u.String(), nil
}

func (s *MinioStorage) GetObject(ctx context.Context, key string) (*minio.Object, error) {
	obj, err := s.client.GetObject(ctx, s.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting object %q: %w", key, err)
	}
	return obj, nil
}

func (s *MinioStorage) CopyObject(ctx context.Context, srcKey, dstKey string) error {
	src := minio.CopySrcOptions{
		Bucket: s.bucketName,
		Object: srcKey,
	}
	dst := minio.CopyDestOptions{
		Bucket: s.bucketName,
		Object: dstKey,
	}
	if _, err := s.client.CopyObject(ctx, dst, src); err != nil {
		return fmt.Errorf("copying object from %q to %q: %w", srcKey, dstKey, err)
	}
	return nil
}

func (s *MinioStorage) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("checking object existence %q: %w", key, err)
	}
	return true, nil
}

func (s *MinioStorage) GetObjectInfo(ctx context.Context, key string) (*ObjectInfo, error) {
	stat, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting object info %q: %w", key, err)
	}
	return &ObjectInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
		LastModified: stat.LastModified,
	}, nil
}
