<template>
  <section>
    <header class="page-head">
      <h1>租户设置</h1>
      <p class="sub">基础信息 + 高级配置</p>
    </header>

    <div class="layout">
      <article class="card">
        <header class="card-head">
          <span class="card-title">基础信息</span>
        </header>
        <form class="form-body" v-if="info" @submit.prevent="save">
          <label class="field">
            <span class="label">租户名称</span>
            <input class="input" v-model="form.name" required />
            <span class="hint">显示在顶部租户切换器中</span>
          </label>
          <label class="field">
            <span class="label">租户 slug</span>
            <input class="input" :value="info.tenant.slug" readonly />
            <span class="hint">不可修改 · 用于 schema 命名</span>
          </label>
          <label class="field">
            <span class="label">数据库 schema</span>
            <input class="input mono" :value="info.tenant.schema_name" readonly />
            <span class="hint">物理隔离每个租户的数据</span>
          </label>
          <label class="field">
            <span class="label">创建时间</span>
            <input class="input" :value="info.tenant.created_at?.slice(0, 19).replace('T', ' ')" readonly />
          </label>
          <div class="form-actions">
            <button class="btn btn-primary" type="submit" :disabled="saving">{{ saving ? '保存中…' : '保存更改' }}</button>
            <span v-if="saved" class="saved-tag">✓ 已保存</span>
          </div>
        </form>
      </article>

      <article class="card stat-card">
        <header class="card-head">
          <span class="card-title">租户状态</span>
        </header>
        <div class="status-body" v-if="info">
          <div class="status-pill" :class="info.tenant.status">
            <span class="dot"></span>
            {{ statusLabel }}
          </div>
          <div class="stat-row">
            <span class="stat-label">成员数</span>
            <span class="stat-val">{{ info.member_count }}</span>
          </div>
          <div class="stat-row">
            <span class="stat-label">已上线应用</span>
            <span class="stat-val">1 / 30+</span>
          </div>
          <div class="stat-row">
            <span class="stat-label">数据隔离</span>
            <span class="stat-val ok">★ Schema 级</span>
          </div>
        </div>
      </article>
    </div>

    <article class="card danger-zone">
      <header class="card-head">
        <span class="card-title danger">危险区</span>
      </header>
      <div class="card-pad">
        <div class="danger-row">
          <div>
            <div class="dr-title">暂停租户</div>
            <div class="dr-sub">所有成员将无法登录到此租户，schema 保留</div>
          </div>
          <button class="btn btn-danger">暂停</button>
        </div>
        <div class="danger-row">
          <div>
            <div class="dr-title">关闭租户</div>
            <div class="dr-sub">30 天保留期后 schema 将被永久删除（演示禁用）</div>
          </div>
          <button class="btn btn-danger" disabled>关闭</button>
        </div>
      </div>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { getTenant, updateTenantName, type TenantInfo } from "../api/admin";

const info = ref<TenantInfo | null>(null);
const form = reactive({ name: "" });
const saving = ref(false);
const saved = ref(false);

const statusLabel = computed(() => {
  const s = info.value?.tenant.status;
  return s === "active" ? "运行中" : s === "suspended" ? "已暂停" : s === "closed" ? "已关闭" : "—";
});

onMounted(async () => {
  info.value = await getTenant();
  form.name = info.value?.tenant.name ?? "";
});

async function save() {
  saving.value = true;
  saved.value = false;
  try {
    await updateTenantName(form.name);
    info.value = await getTenant();
    saved.value = true;
    setTimeout(() => (saved.value = false), 2200);
  } catch (e: any) {
    alert(e.response?.data?.error?.message ?? "保存失败");
  } finally { saving.value = false; }
}
</script>

<style scoped>
.page-head { margin-bottom: 20px; }
.page-head h1 { font-size: 22px; font-weight: 700; }
.page-head .sub { font-size: 13px; color: var(--text-3); margin-top: 4px; }
.layout { display: grid; grid-template-columns: 2fr 1fr; gap: 14px; margin-bottom: 14px; }

.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; }
.card-head { padding: 14px 18px; border-bottom: 1px solid var(--border); }
.card-title { font-size: 14px; font-weight: 600; }
.card-title.danger { color: var(--danger); }

.form-body { padding: 18px; display: flex; flex-direction: column; gap: 16px; }
.field { display: flex; flex-direction: column; gap: 5px; }
.label { font-size: 12.5px; font-weight: 500; color: var(--text-2); }
.input { padding: 8px 11px; border: 1px solid var(--border-strong); border-radius: 7px; font-size: 13px; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.input[readonly] { background: var(--surface-2); color: var(--text-3); }
.input.mono { font-family: var(--ff-mono); font-size: 12px; }
.hint { font-size: 11.5px; color: var(--text-3); }

.form-actions { display: flex; align-items: center; gap: 10px; margin-top: 6px; }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-danger { background: var(--surface); color: var(--danger); border-color: var(--danger-soft); }
.btn-danger:hover { background: var(--danger-soft); }
.btn-danger:disabled { color: var(--text-4); border-color: var(--border); cursor: not-allowed; background: var(--bg); }
.saved-tag { color: var(--success); font-size: 12.5px; font-weight: 600; }

.stat-card .status-body { padding: 18px; display: flex; flex-direction: column; gap: 14px; }
.status-pill {
  display: inline-flex; gap: 8px; align-items: center;
  padding: 6px 12px;
  border-radius: 999px;
  font-size: 13px; font-weight: 600;
  align-self: flex-start;
}
.status-pill.active { background: var(--success-soft); color: var(--success); }
.status-pill.suspended { background: var(--warning-soft); color: var(--warning); }
.status-pill.closed { background: var(--danger-soft); color: var(--danger); }
.status-pill .dot { width: 6px; height: 6px; background: currentColor; border-radius: 999px; }
.stat-row { display: flex; justify-content: space-between; align-items: center; }
.stat-label { font-size: 12.5px; color: var(--text-3); }
.stat-val { font-size: 13px; font-weight: 600; color: var(--text); }
.stat-val.ok { color: var(--success); }

.danger-zone { border-color: var(--danger-soft); }
.danger-row {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 0;
  border-bottom: 1px solid var(--border-soft);
}
.danger-row:last-child { border-bottom: 0; }
.dr-title { font-size: 13.5px; font-weight: 600; }
.dr-sub { font-size: 12px; color: var(--text-3); margin-top: 3px; }
.card-pad { padding: 4px 18px 14px; }
</style>
