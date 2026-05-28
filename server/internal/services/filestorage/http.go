package filestorage

import (
	"io"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// RegisterRoutes wires /files routes.
func RegisterRoutes(r *gin.RouterGroup, svc *Service) {
	r.POST("/files/upload", func(c *gin.Context) {
		claims, _ := iam.ClaimsFromContext(c.Request.Context())
		file, err := c.FormFile("file")
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "filestorage.no_file", "缺少 file 字段", err))
			return
		}
		bizModule := c.PostForm("biz_module")
		bizID := c.PostForm("biz_id")
		if bizModule == "" || bizID == "" {
			apiresp.Fail(c, errors.New(errors.KindParam, "filestorage.missing_biz", "biz_module / biz_id 必填"))
			return
		}
		src, err := file.Open()
		if err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindInternal, "filestorage.read_failed", "读取上传文件失败", err))
			return
		}
		defer src.Close()
		att, err := svc.Upload(c.Request.Context(), UploadCmd{
			BizModule: bizModule,
			BizID:     bizID,
			Name:      file.Filename,
			Size:      file.Size,
			MimeType:  file.Header.Get("Content-Type"),
			Uploader:  claims.MemberID,
			Body:      src,
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, att)
	})

	r.GET("/files/:id/download", func(c *gin.Context) {
		id, err := kernel.ParseID(c.Param("id"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		att, body, err := svc.Download(c.Request.Context(), id)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		defer body.Close()
		c.Header("Content-Disposition", `attachment; filename="`+att.Name+`"`)
		c.Header("Content-Type", att.MimeType)
		c.Header("Content-Length", strconv.FormatInt(att.Size, 10))
		_, _ = io.Copy(c.Writer, body)
	})

	r.GET("/files/biz/:module/:id", func(c *gin.Context) {
		atts, err := svc.ListByBiz(c.Request.Context(), c.Param("module"), c.Param("id"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"attachments": atts})
	})
}
