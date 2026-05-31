<template>
  <div class="news-manage">
    <!-- Left: categories -->
    <aside class="nm-side">
      <div class="nm-side-head">
        <span>栏目管理</span>
        <button class="nm-add" title="新建栏目" @click="openNewCat">＋</button>
      </div>
      <button
        class="nm-cat" :class="{ active: filterCat === '' }"
        @click="filterCat = ''; loadArticles()"
      >
        <span class="nm-cat-label">全部</span>
      </button>
      <div v-for="c in categories" :key="c.id" class="nm-cat-row">
        <button class="nm-cat" :class="{ active: filterCat === c.id }" @click="filterCat = c.id; loadArticles()">
          <span class="nm-cat-label">{{ c.name }}</span>
          <span class="nm-cat-count">{{ c.article_count }}</span>
        </button>
        <div class="nm-cat-actions">
          <button title="重命名" @click="openEditCat(c)">✎</button>
          <button title="删除" class="danger" @click="removeCat(c)">🗑</button>
        </div>
      </div>
    </aside>

    <!-- Right: articles -->
    <main class="nm-main">
      <header class="nm-main-head">
        <input v-model="keyword" class="nm-search" placeholder="搜索标题…" @keyup.enter="loadArticles" />
        <select v-model="statusFilter" class="nm-select" @change="loadArticles">
          <option value="">全部状态</option>
          <option value="draft">草稿</option>
          <option value="published">已发布</option>
        </select>
        <button class="nm-primary" @click="openNewArticle">＋ 新建文章</button>
      </header>

      <div v-if="loading" class="nm-loading">加载中…</div>
      <div v-else-if="articles.length === 0" class="nm-empty">暂无文章</div>
      <table v-else class="nm-table">
        <thead>
          <tr><th>标题</th><th>栏目</th><th>状态</th><th>浏览</th><th>更新时间</th><th>操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="a in articles" :key="a.id">
            <td class="nm-cell-title">{{ a.title }}</td>
            <td>{{ a.category_name || "—" }}</td>
            <td>
              <span class="nm-badge" :class="a.status">{{ a.status === "published" ? "已发布" : "草稿" }}</span>
            </td>
            <td>{{ a.views }}</td>
            <td>{{ fmtDateTime(a.updated_at) }}</td>
            <td class="nm-actions">
              <button @click="openEditArticle(a)">编辑</button>
              <button v-if="a.status === 'draft'" class="ok" @click="doPublish(a)">发布</button>
              <button v-else class="warn" @click="doUnpublish(a)">下线</button>
              <button class="danger" @click="removeArticle(a)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="totalPages > 1" class="nm-pager">
        <button :disabled="page <= 1" @click="page--; loadArticles()">上一页</button>
        <span>{{ page }} / {{ totalPages }}</span>
        <button :disabled="page >= totalPages" @click="page++; loadArticles()">下一页</button>
      </div>
    </main>

    <!-- Category dialog -->
    <div v-if="catDlg.open" class="nm-modal-backdrop" @click.self="catDlg.open = false">
      <div class="nm-modal">
        <h3>{{ catDlg.id ? "编辑栏目" : "新建栏目" }}</h3>
        <label>名称<input v-model="catDlg.name" placeholder="栏目名称" /></label>
        <label>排序<input v-model.number="catDlg.orderNum" type="number" /></label>
        <div class="nm-modal-foot">
          <button @click="catDlg.open = false">取消</button>
          <button class="nm-primary" @click="saveCat">保存</button>
        </div>
      </div>
    </div>

    <!-- Article dialog -->
    <div v-if="artDlg.open" class="nm-modal-backdrop" @click.self="artDlg.open = false">
      <div class="nm-modal nm-modal-lg">
        <h3>{{ artDlg.id ? "编辑文章" : "新建文章" }}</h3>
        <label>标题<input v-model="artDlg.title" placeholder="文章标题" /></label>
        <label>栏目
          <select v-model="artDlg.categoryId">
            <option value="">未分类</option>
            <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
        </label>
        <label>作者<input v-model="artDlg.author" placeholder="作者/来源" /></label>
        <label>封面图 URL<input v-model="artDlg.coverUrl" placeholder="https://…" /></label>
        <label>摘要<textarea v-model="artDlg.summary" rows="2" placeholder="一句话摘要"></textarea></label>
        <label>正文<textarea v-model="artDlg.content" rows="10" placeholder="文章正文"></textarea></label>
        <div class="nm-modal-foot">
          <button @click="artDlg.open = false">取消</button>
          <button class="nm-primary" @click="saveArticle">保存</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import {
  listCategories,
  createCategory,
  updateCategory,
  deleteCategory,
  listArticles,
  createArticle,
  updateArticle,
  publishArticle,
  unpublishArticle,
  deleteArticle,
  type NewsCategory,
  type NewsArticle,
  type ArticleStatus,
} from "../api";

const notify = useNotification();
const { confirm } = useConfirm();

const categories = ref<NewsCategory[]>([]);
const articles = ref<NewsArticle[]>([]);
const filterCat = ref("");
const statusFilter = ref<"" | ArticleStatus>("");
const keyword = ref("");
const page = ref(1);
const pageSize = 20;
const total = ref(0);
const loading = ref(false);

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)));

const catDlg = reactive({ open: false, id: "", name: "", orderNum: 0 });
const artDlg = reactive({
  open: false,
  id: "",
  title: "",
  categoryId: "",
  author: "",
  coverUrl: "",
  summary: "",
  content: "",
});

function fmtDateTime(s?: string | null): string {
  if (!s) return "";
  const d = new Date(s);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
}

async function loadCategories() {
  try {
    categories.value = await listCategories();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载栏目失败");
  }
}

async function loadArticles() {
  loading.value = true;
  try {
    const r = await listArticles({
      category_id: filterCat.value || undefined,
      status: statusFilter.value || undefined,
      keyword: keyword.value || undefined,
      page: page.value,
      page_size: pageSize,
    });
    articles.value = r.items;
    total.value = r.total;
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载文章失败");
  } finally {
    loading.value = false;
  }
}

// ---- Category dialog ----
function openNewCat() {
  catDlg.open = true;
  catDlg.id = "";
  catDlg.name = "";
  catDlg.orderNum = 0;
}
function openEditCat(c: NewsCategory) {
  catDlg.open = true;
  catDlg.id = c.id;
  catDlg.name = c.name;
  catDlg.orderNum = c.order_num;
}
async function saveCat() {
  if (!catDlg.name.trim()) {
    notify.error("栏目名称不能为空");
    return;
  }
  try {
    if (catDlg.id) {
      await updateCategory(catDlg.id, { name: catDlg.name, order_num: catDlg.orderNum });
    } else {
      await createCategory(catDlg.name, catDlg.orderNum);
    }
    catDlg.open = false;
    await loadCategories();
    notify.success("已保存");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "保存失败");
  }
}
async function removeCat(c: NewsCategory) {
  if (!(await confirm({ title: "删除栏目", message: `确定删除栏目「${c.name}」？文章将变为未分类。`, danger: true }))) return;
  try {
    await deleteCategory(c.id);
    if (filterCat.value === c.id) filterCat.value = "";
    await loadCategories();
    await loadArticles();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}

// ---- Article dialog ----
function openNewArticle() {
  artDlg.open = true;
  artDlg.id = "";
  artDlg.title = "";
  artDlg.categoryId = filterCat.value || "";
  artDlg.author = "";
  artDlg.coverUrl = "";
  artDlg.summary = "";
  artDlg.content = "";
}
function openEditArticle(a: NewsArticle) {
  artDlg.open = true;
  artDlg.id = a.id;
  artDlg.title = a.title;
  artDlg.categoryId = a.category_id ?? "";
  artDlg.author = a.author;
  artDlg.coverUrl = a.cover_url;
  artDlg.summary = a.summary;
  artDlg.content = a.content;
}
async function saveArticle() {
  if (!artDlg.title.trim()) {
    notify.error("标题不能为空");
    return;
  }
  const payload = {
    title: artDlg.title,
    category_id: artDlg.categoryId,
    author: artDlg.author,
    cover_url: artDlg.coverUrl,
    summary: artDlg.summary,
    content: artDlg.content,
  };
  try {
    if (artDlg.id) {
      await updateArticle(artDlg.id, payload);
    } else {
      await createArticle(payload);
    }
    artDlg.open = false;
    await loadArticles();
    await loadCategories();
    notify.success("已保存");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "保存失败");
  }
}
async function doPublish(a: NewsArticle) {
  try {
    await publishArticle(a.id);
    await loadArticles();
    notify.success("已发布");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "发布失败");
  }
}
async function doUnpublish(a: NewsArticle) {
  try {
    await unpublishArticle(a.id);
    await loadArticles();
    notify.success("已下线");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "操作失败");
  }
}
async function removeArticle(a: NewsArticle) {
  if (!(await confirm({ title: "删除文章", message: `确定删除「${a.title}」？`, danger: true }))) return;
  try {
    await deleteArticle(a.id);
    await loadArticles();
    await loadCategories();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}

onMounted(() => {
  loadCategories();
  loadArticles();
});
</script>

<style scoped>
.news-manage { display: flex; height: 100%; gap: 16px; }

.nm-side {
  width: 220px;
  flex-shrink: 0;
  border-right: 1px solid var(--border, #e5e7eb);
  padding: 12px 8px;
  overflow-y: auto;
}
.nm-side-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: var(--text-4, #9ca3af);
  padding: 4px 10px 8px;
  letter-spacing: 0.05em;
}
.nm-add, .nm-cat-actions button {
  border: none;
  background: transparent;
  cursor: pointer;
  color: var(--text-3, #6b7280);
  font-size: 13px;
}
.nm-cat-row { display: flex; align-items: center; }
.nm-cat-row:hover .nm-cat-actions { opacity: 1; }
.nm-cat-actions { opacity: 0; display: flex; gap: 2px; transition: opacity 0.15s; }
.nm-cat-actions .danger { color: #dc2626; }
.nm-cat {
  display: flex;
  align-items: center;
  flex: 1;
  gap: 8px;
  padding: 8px 10px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  text-align: left;
  color: var(--text-1, #1f2937);
  font-size: 14px;
}
.nm-cat:hover { background: var(--bg-2, #f3f4f6); }
.nm-cat.active { background: var(--cat-content-soft, #fff0ea); color: var(--cat-content, #d14f27); font-weight: 600; }
.nm-cat-label { flex: 1; }
.nm-cat-count { font-size: 12px; color: var(--text-4, #9ca3af); }

.nm-main { flex: 1; overflow-y: auto; padding: 12px 16px; }
.nm-main-head { display: flex; gap: 10px; align-items: center; margin-bottom: 14px; }
.nm-search, .nm-select {
  padding: 7px 10px;
  border: 1px solid var(--border, #d1d5db);
  border-radius: 8px;
  font-size: 14px;
}
.nm-search { flex: 1; max-width: 280px; }
.nm-primary {
  margin-left: auto;
  padding: 7px 16px;
  border: none;
  background: var(--cat-content, #d14f27);
  color: #fff;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
}

.nm-table { width: 100%; border-collapse: collapse; font-size: 14px; }
.nm-table th, .nm-table td { text-align: left; padding: 10px 8px; border-bottom: 1px solid var(--border, #eee); }
.nm-table th { font-size: 12px; color: var(--text-4, #9ca3af); font-weight: 500; }
.nm-cell-title { font-weight: 600; max-width: 320px; }
.nm-badge { padding: 2px 10px; border-radius: 10px; font-size: 12px; }
.nm-badge.published { background: #dcfce7; color: #166534; }
.nm-badge.draft { background: #f3f4f6; color: #6b7280; }
.nm-actions { white-space: nowrap; }
.nm-actions button {
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 13px;
  margin-right: 8px;
  color: var(--cat-content, #d14f27);
}
.nm-actions .ok { color: #16a34a; }
.nm-actions .warn { color: #d97706; }
.nm-actions .danger { color: #dc2626; }

.nm-pager { display: flex; gap: 12px; align-items: center; justify-content: center; padding: 16px; }
.nm-pager button { padding: 6px 14px; border: 1px solid var(--border, #d1d5db); background: #fff; border-radius: 8px; cursor: pointer; }
.nm-pager button:disabled { opacity: 0.45; cursor: not-allowed; }

.nm-loading, .nm-empty { padding: 48px; text-align: center; color: var(--text-4, #9ca3af); }

.nm-modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.nm-modal {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  width: 420px;
  max-width: 92vw;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.nm-modal-lg { width: 640px; max-height: 88vh; overflow-y: auto; }
.nm-modal h3 { margin: 0 0 4px; }
.nm-modal label { display: flex; flex-direction: column; gap: 4px; font-size: 13px; color: var(--text-3, #6b7280); }
.nm-modal input, .nm-modal textarea, .nm-modal select {
  padding: 8px 10px;
  border: 1px solid var(--border, #d1d5db);
  border-radius: 8px;
  font-size: 14px;
  font-family: inherit;
}
.nm-modal textarea { resize: vertical; }
.nm-modal-foot { display: flex; justify-content: flex-end; gap: 10px; margin-top: 8px; }
.nm-modal-foot button {
  padding: 8px 18px;
  border: 1px solid var(--border, #d1d5db);
  background: #fff;
  border-radius: 8px;
  cursor: pointer;
}
.nm-modal-foot .nm-primary { border: none; background: var(--cat-content, #d14f27); color: #fff; }
</style>
