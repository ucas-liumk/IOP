package filestorage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

const (
	MaxFileSize   = 50 << 20 // 50 MB
	defaultBucket = "iop-files"
)

type Attachment struct {
	ID        kernel.ID `json:"id"`
	BizModule string    `json:"biz_module"`
	BizID     string    `json:"biz_id"`
	ObjectKey string    `json:"object_key"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	MimeType  string    `json:"mime_type"`
	Uploader  kernel.ID `json:"uploader"`
	CreatedAt time.Time `json:"created_at"`
}

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
}

// Service uploads to MinIO + persists Attachment metadata in tenant schema.
type Service struct {
	tenant *tenantdb.TenantDB
	mc     *minio.Client
	bucket string
	clock  kernel.Clock
}

func NewService(ctx context.Context, t *tenantdb.TenantDB, cfg Config, clk kernel.Clock) (*Service, error) {
	if cfg.Bucket == "" {
		cfg.Bucket = defaultBucket
	}
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, errors.Wrap(errors.KindExternal, "filestorage.minio_init", "minio init failed", err)
	}
	// Best-effort bucket creation (idempotent).
	exists, _ := mc.BucketExists(ctx, cfg.Bucket)
	if !exists {
		if err := mc.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			// ignore — minio sometimes races; second call will work
		}
	}
	return &Service{tenant: t, mc: mc, bucket: cfg.Bucket, clock: clk}, nil
}

type UploadCmd struct {
	BizModule string
	BizID     string
	Name      string
	Size      int64
	MimeType  string
	Uploader  kernel.ID
	Body      io.Reader
}

func (s *Service) Upload(ctx context.Context, cmd UploadCmd) (*Attachment, error) {
	if cmd.Size > MaxFileSize {
		return nil, errors.New(errors.KindParam, "filestorage.too_large",
			fmt.Sprintf("文件超出限制 (max %d bytes)", MaxFileSize))
	}
	tc, ok := tenantdb.FromContext(ctx)
	if !ok {
		return nil, errors.New(errors.KindInternal, "filestorage.no_tenant", "no tenant ctx")
	}
	att := &Attachment{
		ID:        kernel.NewID(),
		BizModule: cmd.BizModule,
		BizID:     cmd.BizID,
		Name:      cmd.Name,
		Size:      cmd.Size,
		MimeType:  cmd.MimeType,
		Uploader:  cmd.Uploader,
		CreatedAt: s.clock.Now(),
	}
	att.ObjectKey = fmt.Sprintf("%s/%s/%s/%s", tc.Slug, cmd.BizModule, cmd.BizID, att.ID)

	if _, err := s.mc.PutObject(ctx, s.bucket, att.ObjectKey, cmd.Body, cmd.Size, minio.PutObjectOptions{
		ContentType: cmd.MimeType,
	}); err != nil {
		return nil, errors.Wrap(errors.KindExternal, "filestorage.upload_failed", "minio upload failed", err)
	}
	if err := s.tenant.Transaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO attachment (id, biz_module, biz_id, object_key, name, size, mime_type, uploader, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			att.ID, att.BizModule, att.BizID, att.ObjectKey, att.Name, att.Size, att.MimeType, att.Uploader, att.CreatedAt)
		return err
	}); err != nil {
		// Best-effort cleanup
		_ = s.mc.RemoveObject(ctx, s.bucket, att.ObjectKey, minio.RemoveObjectOptions{})
		return nil, errors.Wrap(errors.KindDatabase, "filestorage.persist_failed", "attachment insert failed", err)
	}
	return att, nil
}

func (s *Service) Download(ctx context.Context, id kernel.ID) (*Attachment, io.ReadCloser, error) {
	var att *Attachment
	err := s.tenant.Transaction(ctx, func(tx pgx.Tx) error {
		a := &Attachment{}
		err := tx.QueryRow(ctx,
			`SELECT id, biz_module, biz_id, object_key, name, size, COALESCE(mime_type,''), uploader, created_at
			 FROM attachment WHERE id = $1`, id).
			Scan(&a.ID, &a.BizModule, &a.BizID, &a.ObjectKey, &a.Name, &a.Size, &a.MimeType, &a.Uploader, &a.CreatedAt)
		if err != nil {
			return err
		}
		att = a
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	obj, err := s.mc.GetObject(ctx, s.bucket, att.ObjectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, errors.Wrap(errors.KindExternal, "filestorage.download_failed", "minio download failed", err)
	}
	return att, obj, nil
}

func (s *Service) ListByBiz(ctx context.Context, module, id string) ([]*Attachment, error) {
	var out []*Attachment
	err := s.tenant.Transaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, biz_module, biz_id, object_key, name, size, COALESCE(mime_type,''), uploader, created_at
			 FROM attachment WHERE biz_module = $1 AND biz_id = $2 ORDER BY created_at DESC`,
			module, id)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			a := &Attachment{}
			if err := rows.Scan(&a.ID, &a.BizModule, &a.BizID, &a.ObjectKey, &a.Name, &a.Size, &a.MimeType, &a.Uploader, &a.CreatedAt); err != nil {
				return err
			}
			out = append(out, a)
		}
		return rows.Err()
	})
	return out, err
}
