<!--
感谢贡献！请填写下面的内容并勾选核对清单。
Thanks for contributing! Please fill this in and complete the checklist.
-->

## 变更说明 / Summary

简要说明这个 PR 做了什么，以及「为什么」。
Briefly describe what this PR does and **why**.

## 关联 issue / Related issues

<!-- e.g. Closes #123 -->

## 变更类型 / Type of change

- [ ] feat（新功能 / new feature）
- [ ] fix（缺陷修复 / bug fix）
- [ ] refactor（重构 / refactor）
- [ ] docs（文档 / documentation）
- [ ] chore / test / 其他 other

## 核对清单 / Checklist

- [ ] **测试通过 / Tests pass** — 后端 `make -C server test` 与 `IOP_INTEGRATION=1 go test ./...`（含命门隔离测试）；前端 `npm run build`（含 `vue-tsc` 类型检查）。
- [ ] **Lint 干净 / Lint clean** — `gofmt` + `go vet` + `make -C server lint`（`golangci-lint`）无告警；前端 ESLint（若配置）无告警。
- [ ] **文档已更新 / Docs updated** — 涉及行为/接口/配置变更时已更新 README、`docs/`、必要时 `CHANGELOG.md` 的 Unreleased 段。
- [ ] **无密钥泄露 / No secrets** — 未提交 JWT secret、口令、token、`.env` 等敏感信息；日志未打印密钥（`logger.Sanitize`）。
- [ ] **已考虑 RBAC / RBAC considered** — 新增端点已声明并校验权限（resource × action），并复核了多租户隔离（租户级访问走 `tenantdb.TenantDB.Transaction`，无跨租户数据可达）。
- [ ] 提交信息遵循约定（Conventional-Commits 风格，必要时附 `Co-Authored-By`）。

## 附加说明 / Notes for reviewers

<!-- 截图、迁移注意事项、回滚方式等 / screenshots, migration notes, rollback plan -->
