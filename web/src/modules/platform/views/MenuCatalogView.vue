<template>
  <section class="admin-page">
    <PageHeader title="菜单管理" :sub="`平台菜单目录 · ${rows.length} 个节点`">
      <template #actions>
        <button class="btn btn-ghost" @click="reload">刷新</button>
        <button class="btn btn-primary" v-perm="'menu:write'" @click="openCreate('dir')">+ 新增目录</button>
        <button class="btn btn-primary" v-perm="'menu:write'" @click="openCreate('menu')">+ 新增菜单</button>
        <button class="btn btn-primary" v-perm="'menu:write'" @click="openCreate('button')">+ 新增按钮</button>
      </template>
    </PageHeader>

    <article class="card">
      <table class="tree-table">
        <thead>
          <tr>
            <th>菜单名称</th>
            <th>类型</th>
            <th>路由 / 组件</th>
            <th>权限标识</th>
            <th>应用</th>
            <th>租户类型</th>
            <th>状态</th>
            <th>排序</th>
            <th class="op">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.key">
            <td>
              <div class="name-cell" :style="{ paddingLeft: `${row.depth * 18}px` }">
                <span class="tree-mark">{{ row.children?.length ? '▾' : '' }}</span>
                <svg v-if="row.icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path :d="row.icon" />
                </svg>
                <span class="menu-title">{{ row.title }}</span>
                <code>{{ row.key }}</code>
                <em v-if="row.built_in">内置</em>
              </div>
            </td>
            <td><span class="type-tag">{{ typeLabel(row.type) }}</span></td>
            <td>
              <div class="path-cell">
                <code v-if="row.path">{{ row.path }}</code>
                <small v-if="row.component">{{ row.component }}</small>
                <small v-if="row.external_url">{{ row.external_url }}</small>
                <small v-if="row.iframe_url">{{ row.iframe_url }}</small>
              </div>
            </td>
            <td><code v-if="row.perm" class="perm">{{ row.perm }}</code><span v-else class="muted">-</span></td>
            <td><code v-if="row.app">{{ row.app }}</code><span v-else class="muted">内置</span></td>
            <td>{{ consoleLabel(row.console) }}</td>
            <td>
              <span class="status" :class="row.status === 'active' ? 'on' : 'off'">{{ row.status === 'active' ? '正常' : '停用' }}</span>
              <span v-if="row.visible === false" class="status off">隐藏</span>
            </td>
            <td>{{ row.order }}</td>
            <td class="op">
              <button class="link-btn" v-perm="'menu:write'" @click="openCreate('menu', row.key)">加子级</button>
              <button class="link-btn" v-perm="'menu:write'" @click="openEdit(row)">编辑</button>
              <button class="link-btn danger" v-if="!row.built_in" v-perm="'menu:write'" @click="remove(row)">删除</button>
            </td>
          </tr>
          <tr v-if="rows.length === 0"><td colspan="9" class="empty">暂无菜单</td></tr>
        </tbody>
      </table>
    </article>

    <div v-if="editing" class="modal-overlay" @click.self="closeEditor">
      <div class="modal">
        <h3>{{ isNew ? '新增菜单' : '编辑菜单' }}</h3>
        <div class="form-grid">
          <label class="field">
            <span>菜单标识 *</span>
            <input class="input mono" v-model="form.key" :disabled="!isNew" placeholder="例如 platform.reports" />
          </label>
          <label class="field">
            <span>菜单名称 *</span>
            <input class="input" v-model="form.title" />
          </label>
          <label class="field">
            <span>菜单类型</span>
            <select class="input" v-model="form.type">
              <option v-for="t in typeOptions" :key="t.value" :value="t.value">{{ t.label }}</option>
            </select>
          </label>
          <label class="field">
            <span>父级菜单</span>
            <select class="input" v-model="form.parent">
              <option value="">顶级</option>
              <option v-for="p in parentOptions" :key="p.key" :value="p.key">{{ '　'.repeat(p.depth) }}{{ p.title }}</option>
            </select>
          </label>
          <label class="field">
            <span>路由路径</span>
            <input class="input mono" v-model="form.path" placeholder="/platform/example" />
          </label>
          <label class="field">
            <span>组件路径</span>
            <input class="input mono" v-model="form.component" placeholder="@/modules/..." />
          </label>
          <label class="field">
            <span>权限标识</span>
            <input class="input mono" v-model="form.permission_code" placeholder="user:add" />
          </label>
          <label class="field">
            <span>所属应用</span>
            <input class="input mono" v-model="form.app_code" placeholder="app_code" />
          </label>
          <label class="field">
            <span>租户类型</span>
            <select class="input" v-model="form.console">
              <option value="platform">平台</option>
              <option value="tenant">租户</option>
              <option value="both">通用</option>
            </select>
          </label>
          <label class="field">
            <span>排序</span>
            <input class="input" type="number" v-model.number="form.order" />
          </label>
          <label class="field">
            <span>状态</span>
            <select class="input" v-model="form.status">
              <option value="active">正常</option>
              <option value="disabled">停用</option>
            </select>
          </label>
          <label class="field check-field">
            <input type="checkbox" v-model="form.visible" />
            <span>显示</span>
            <input type="checkbox" v-model="form.cacheable" />
            <span>缓存</span>
          </label>
        </div>
        <div class="icon-picker">
          <button v-for="i in iconPresets" :key="i" type="button" class="icon-btn" :class="{ picked: form.icon === i }" @click="form.icon = i">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path :d="i" /></svg>
          </button>
          <input class="input mono icon-input" v-model="form.icon" placeholder="SVG path" />
        </div>
        <div class="form-grid">
          <label class="field"><span>外链 URL</span><input class="input mono" v-model="form.external_url" /></label>
          <label class="field"><span>iframe URL</span><input class="input mono" v-model="form.iframe_url" /></label>
          <label class="field"><span>微前端应用</span><input class="input mono" v-model="form.micro_app_code" /></label>
          <label class="field"><span>微前端入口</span><input class="input mono" v-model="form.micro_entry" /></label>
        </div>
        <div v-if="formError" class="form-error">{{ formError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="closeEditor">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="save">保存</button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { PageHeader } from "@/shell/components";
import { useConfirm } from "@/shell/confirm";
import { useNotification } from "@/shell/notify";
import {
  createPlatformMenu, deletePlatformMenu, getPlatformMenuTree, updatePlatformMenu,
  type MenuPayload, type MenuTreeNode,
} from "../api/menus";

type Row = MenuTreeNode & { depth: number };

const notify = useNotification();
const { confirm } = useConfirm();
const menus = ref<MenuTreeNode[]>([]);
const editing = ref(false);
const isNew = ref(true);
const editKey = ref("");
const busy = ref(false);
const formError = ref("");

const typeOptions = [
  { value: "dir", label: "目录" },
  { value: "menu", label: "菜单" },
  { value: "button", label: "按钮" },
  { value: "link", label: "外链" },
  { value: "iframe", label: "iframe" },
  { value: "micro", label: "微前端" },
];
const iconPresets = [
  "M3 12h18M3 6h18M3 18h18",
  "M3 3h7v9H3zM14 3h7v5h-7zM14 12h7v9h-7zM3 16h7v5H3z",
  "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z",
  "M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z",
  "M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2",
];

const form = reactive<MenuPayload>({
  key: "", title: "", type: "menu", parent: "", path: "", component: "",
  permission_code: "", icon: iconPresets[0], order: 0, visible: true,
  cacheable: false, status: "active", app_code: "", console: "platform",
  external_url: "", iframe_url: "", micro_app_code: "", micro_entry: "",
});

const rows = computed<Row[]>(() => flatten(menus.value));
const parentOptions = computed(() => rows.value.filter((r) => r.type !== "button" && r.key !== editKey.value));

onMounted(reload);

async function reload() {
  menus.value = await getPlatformMenuTree();
}
function flatten(nodes: MenuTreeNode[], depth = 0): Row[] {
  const out: Row[] = [];
  for (const n of nodes) {
    out.push({ ...n, depth });
    if (n.children?.length) out.push(...flatten(n.children, depth + 1));
  }
  return out;
}
function openCreate(type: string, parent = "") {
  isNew.value = true;
  editKey.value = "";
  formError.value = "";
  Object.assign(form, {
    key: "", title: "", type, parent, path: "", component: "", permission_code: "",
    icon: iconPresets[0], order: rows.value.length + 1, visible: true, cacheable: false,
    status: "active", app_code: "", console: parent ? (rows.value.find((r) => r.key === parent)?.console ?? "platform") : "platform",
    external_url: "", iframe_url: "", micro_app_code: "", micro_entry: "",
  });
  editing.value = true;
}
function openEdit(row: Row) {
  isNew.value = false;
  editKey.value = row.key;
  formError.value = "";
  Object.assign(form, {
    key: row.key, title: row.title, type: row.type, parent: row.parent ?? "",
    path: row.path ?? "", component: row.component ?? "", permission_code: row.perm ?? "",
    icon: row.icon ?? "", order: row.order ?? 0, visible: row.visible !== false,
    cacheable: !!row.cacheable, status: row.status ?? "active", app_code: row.app ?? "",
    console: row.console ?? "platform", external_url: row.external_url ?? "",
    iframe_url: row.iframe_url ?? "", micro_app_code: row.micro_app_code ?? "", micro_entry: row.micro_entry ?? "",
  });
  editing.value = true;
}
function closeEditor() { editing.value = false; }
async function save() {
  if (!form.key.trim() || !form.title.trim()) { formError.value = "菜单标识和名称必填"; return; }
  if (form.type === "button" && !form.permission_code?.trim()) { formError.value = "按钮必须配置权限标识"; return; }
  busy.value = true; formError.value = "";
  try {
    const payload = { ...form, key: form.key.trim(), title: form.title.trim(), permission_code: form.permission_code?.trim() };
    if (isNew.value) await createPlatformMenu(payload);
    else await updatePlatformMenu(editKey.value, payload);
    editing.value = false;
    await reload();
    notify.success("菜单已保存");
  } catch (e: any) {
    formError.value = e.response?.data?.error?.message ?? "保存失败";
  } finally { busy.value = false; }
}
async function remove(row: Row) {
  if (!(await confirm({ title: "删除菜单", message: `确认删除「${row.title}」？`, danger: true }))) return;
  try {
    await deletePlatformMenu(row.key);
    await reload();
    notify.success("菜单已删除");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}
function typeLabel(t: string) {
  return Object.fromEntries(typeOptions.map((x) => [x.value, x.label]))[t] ?? t;
}
function consoleLabel(v: string) {
  return ({ platform: "平台", tenant: "租户", both: "通用" } as Record<string, string>)[v] ?? v;
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; overflow: hidden; }
.tree-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.tree-table th, .tree-table td { padding: 9px 10px; border-bottom: 1px solid var(--border-soft); text-align: left; vertical-align: top; }
.tree-table th { font-size: 11.5px; color: var(--text-3); font-weight: 700; text-transform: uppercase; letter-spacing: .4px; }
.op { width: 150px; white-space: nowrap; }
.name-cell { display: flex; align-items: center; gap: 7px; min-width: 260px; }
.tree-mark { width: 12px; color: var(--text-4); }
.menu-title { font-weight: 600; }
code { font-family: var(--ff-mono); font-size: 11px; color: var(--text-3); }
em { font-style: normal; font-size: 10px; color: var(--purple); background: var(--purple-soft); padding: 1px 5px; border-radius: 3px; }
.type-tag, .status { font-size: 11px; font-weight: 700; padding: 2px 6px; border-radius: 4px; background: var(--surface-2); color: var(--text-2); }
.status.on { background: var(--success-soft); color: var(--success); }
.status.off { background: var(--surface-2); color: var(--text-3); margin-left: 4px; }
.path-cell { display: flex; flex-direction: column; gap: 2px; }
.path-cell small { color: var(--text-4); font-family: var(--ff-mono); }
.perm { color: var(--primary); background: var(--primary-soft); padding: 1px 5px; border-radius: 3px; }
.muted, .empty { color: var(--text-4); }
.empty { text-align: center; padding: 24px; }
.link-btn { border: 0; background: transparent; color: var(--primary); cursor: pointer; font-size: 12px; padding: 3px 5px; }
.link-btn:hover { background: var(--primary-soft); border-radius: 4px; }
.link-btn.danger { color: var(--danger); }
.link-btn.danger:hover { background: var(--danger-soft); }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-ghost { background: transparent; }
.modal-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.modal { width: min(820px, 94vw); background: var(--surface); border-radius: 12px; padding: 22px; box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 12px; }
.modal h3 { margin: 0; font-size: 16px; }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.field { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.field span { font-size: 12px; color: var(--text-2); }
.input { padding: 7px 10px; border: 1px solid var(--border-strong); border-radius: 6px; background: var(--surface); font-size: 13px; min-width: 0; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.mono { font-family: var(--ff-mono); }
.check-field { flex-direction: row; align-items: center; gap: 8px; padding-top: 20px; }
.icon-picker { display: flex; align-items: center; gap: 8px; }
.icon-btn { width: 32px; height: 30px; display: grid; place-items: center; border: 1px solid var(--border); background: var(--surface); border-radius: 6px; cursor: pointer; }
.icon-btn.picked { border-color: var(--primary); color: var(--primary); background: var(--primary-soft); }
.icon-input { flex: 1; }
.form-error { color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; font-size: 12.5px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 8px; }
@media (max-width: 900px) { .form-grid { grid-template-columns: 1fr; } }
</style>
