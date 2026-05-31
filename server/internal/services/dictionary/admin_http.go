package dictionary

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

// AdminConfig is what RegisterAdminRoutes needs: the in-memory seed (so we know
// all platform-default dict types) and the tenantDB for override CRUD.
type AdminConfig struct {
	Memory   Repository
	TenantDB *tenantdb.TenantDB
}

type AuthzFunc func(resource, action string) gin.HandlerFunc

// RegisterAdminRoutes adds /admin/dict listing + per-tenant override editing.
func RegisterAdminRoutes(r *gin.RouterGroup, cfg AdminConfig, allTypes []string, authz AuthzFunc) {
	r.GET("/admin/dict/types", authz("dict", "read"), func(c *gin.Context) {
		apiresp.OK(c, gin.H{"types": allTypes})
	})

	r.GET("/admin/dict/:typeCode/items", authz("dict", "read"), func(c *gin.Context) {
		typeCode := c.Param("typeCode")
		// Platform default
		baseItems, err := cfg.Memory.List(c.Request.Context(), typeCode)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		// Tenant overrides
		overrides := map[string]map[string]any{}
		err = cfg.TenantDB.Transaction(c.Request.Context(), func(tx pgx.Tx) error {
			rows, err := tx.Query(c.Request.Context(),
				`SELECT item_code, override FROM dict_override WHERE type_code = $1`, typeCode)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var code string
				var raw []byte
				if err := rows.Scan(&code, &raw); err != nil {
					return err
				}
				o := map[string]any{}
				_ = jsonScan(raw, &o)
				overrides[code] = o
			}
			return rows.Err()
		})
		_ = err
		apiresp.OK(c, gin.H{
			"type_code": typeCode,
			"items":     baseItems,
			"overrides": overrides,
		})
	})

	r.PUT("/admin/dict/:typeCode/items/:code/override", authz("dict", "write"), func(c *gin.Context) {
		typeCode := c.Param("typeCode")
		code := c.Param("code")
		var req struct {
			Name      string `json:"name"`
			SortOrder int    `json:"sort_order"`
			Active    bool   `json:"active"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "dict.invalid_request", "请求格式错误", err))
			return
		}
		payload := map[string]any{
			"name":       req.Name,
			"sort_order": req.SortOrder,
			"active":     req.Active,
		}
		raw, _ := jsonMarshal(payload)
		err := cfg.TenantDB.Transaction(c.Request.Context(), func(tx pgx.Tx) error {
			_, err := tx.Exec(c.Request.Context(),
				`INSERT INTO dict_override (type_code, item_code, override, updated_at)
				 VALUES ($1, $2, $3::jsonb, now())
				 ON CONFLICT (type_code, item_code) DO UPDATE
				   SET override = EXCLUDED.override, updated_at = now()`,
				typeCode, code, string(raw))
			return err
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})

	r.DELETE("/admin/dict/:typeCode/items/:code/override", authz("dict", "write"), func(c *gin.Context) {
		typeCode := c.Param("typeCode")
		code := c.Param("code")
		err := cfg.TenantDB.Transaction(c.Request.Context(), func(tx pgx.Tx) error {
			_, err := tx.Exec(c.Request.Context(),
				`DELETE FROM dict_override WHERE type_code = $1 AND item_code = $2`,
				typeCode, code)
			return err
		})
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"ok": true})
	})
}

// jsonMarshal/jsonScan kept tiny to avoid pulling encoding/json in calls above.
func jsonMarshal(v any) ([]byte, error) {
	return jsonImpl.marshal(v)
}
func jsonScan(b []byte, v any) error {
	return jsonImpl.unmarshal(b, v)
}
