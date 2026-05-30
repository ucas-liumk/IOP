package iam

import (
	"context"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// ScopeSpec is the resolved data-access scope for a member within a tenant.
//
// Business modules use this helper through module.DataScopeFunc to apply
// row-level visibility in addition to route-level RBAC:
//
//	spec, _ := iam.ResolveDataScope(ctx, memberID, tenantID)
//	switch spec.Kind {
//	case iam.DataScopeAll:        // no filter
//	case iam.DataScopeSelf:       // WHERE owner_member_id = spec.SelfMemberID
//	case iam.DataScopeDept,
//	     iam.DataScopeDeptAndSub, // WHERE dept_id = ANY(spec.DeptIDs)
//	     iam.DataScopeCustom:     // WHERE dept_id = ANY(spec.DeptIDs)
//	}
//
// For dept / dept_and_sub the concrete dept id set must be supplied by the caller
// (it lives in the tenant schema's department table; this package is tenant-less),
// so DeptIDs is only pre-populated for the "custom" scope (from public.role_dept).
type ScopeSpec struct {
	Kind         string      // all | dept | dept_and_sub | self | custom
	DeptIDs      []kernel.ID // populated for custom; for dept/dept_and_sub the caller expands from the member's own dept
	SelfMemberID kernel.ID
}

// scopeRank orders scopes from most permissive (highest) to least. "custom" is
// treated as its own kind and only wins when no broader plain scope is present.
func scopeRank(kind string) int {
	switch kind {
	case DataScopeAll:
		return 5
	case DataScopeDeptAndSub:
		return 4
	case DataScopeDept:
		return 3
	case DataScopeCustom:
		return 2
	case DataScopeSelf:
		return 1
	}
	return 0
}

// ResolveDataScope returns the most-permissive data scope across all roles granted
// to the member in the tenant. Precedence: all > dept_and_sub > dept > custom > self.
// When the winning scope is "custom", DeptIDs is the UNION of the member's roles'
// public.role_dept bindings for this tenant. Defaults to "self" when the member has
// no roles (least privilege).
func (s *Service) ResolveDataScope(ctx context.Context, memberID, tenantID kernel.ID) (ScopeSpec, error) {
	pool := s.repo.(*pgRepo).pool

	rows, err := pool.Query(ctx,
		`SELECT r.id, r.data_scope
		 FROM public.role_grant g
		 JOIN public.role r ON r.id = g.role_id
		 WHERE g.member_id = $1 AND g.tenant_id = $2 AND r.deleted_at IS NULL`, memberID, tenantID)
	if err != nil {
		return ScopeSpec{}, err
	}
	defer rows.Close()

	bestKind := ""
	bestRank := -1
	customRoleIDs := []kernel.ID{}
	for rows.Next() {
		var rid kernel.ID
		var scope string
		if err := rows.Scan(&rid, &scope); err != nil {
			return ScopeSpec{}, err
		}
		if scope == DataScopeCustom {
			customRoleIDs = append(customRoleIDs, rid)
		}
		if scopeRank(scope) > bestRank {
			bestRank = scopeRank(scope)
			bestKind = scope
		}
	}
	if err := rows.Err(); err != nil {
		return ScopeSpec{}, err
	}

	if bestKind == "" {
		// No roles at all → least privilege.
		return ScopeSpec{Kind: DataScopeSelf, SelfMemberID: memberID}, nil
	}

	spec := ScopeSpec{Kind: bestKind, SelfMemberID: memberID}
	if bestKind == DataScopeCustom && len(customRoleIDs) > 0 {
		drows, err := pool.Query(ctx,
			`SELECT DISTINCT dept_id FROM public.role_dept
			 WHERE tenant_id = $1 AND role_id = ANY($2)`, tenantID, customRoleIDs)
		if err != nil {
			return ScopeSpec{}, err
		}
		defer drows.Close()
		for drows.Next() {
			var did kernel.ID
			if err := drows.Scan(&did); err != nil {
				return ScopeSpec{}, err
			}
			spec.DeptIDs = append(spec.DeptIDs, did)
		}
		if err := drows.Err(); err != nil {
			return ScopeSpec{}, err
		}
	}
	return spec, nil
}
