<template>
  <div class="bkm-app">
    <header class="bkm-head">
      <div class="bkm-tabs">
        <button class="bkm-tab" :class="{ active: tab === 'books' }" @click="tab = 'books'">图书管理</button>
        <button class="bkm-tab" :class="{ active: tab === 'borrows' }" @click="switchToBorrows">借阅记录</button>
      </div>
      <div class="bkm-actions">
        <template v-if="tab === 'books'">
          <input
            v-model="search" class="bkm-input" placeholder="搜索书名 / 作者 / ISBN"
            @keyup.enter="doSearch"
          />
          <button class="btn" @click="doSearch">搜索</button>
          <button class="btn btn-primary" @click="openCreate">＋ 新增图书</button>
        </template>
        <template v-else>
          <select v-model="borrowFilter" class="bkm-input" @change="loadBorrows">
            <option value="">全部状态</option>
            <option value="borrowed">借阅中</option>
            <option value="returned">已归还</option>
          </select>
        </template>
      </div>
    </header>

    <!-- Book management -->
    <section v-if="tab === 'books'">
      <div v-if="loading" class="bkm-empty">加载中…</div>
      <div v-else-if="books.length === 0" class="bkm-empty">还没有图书，点击「新增图书」开始</div>
      <table v-else class="bkm-table">
        <thead>
          <tr>
            <th>书名</th><th>作者</th><th>ISBN</th><th>分类</th><th>馆藏位置</th><th class="num">可借/总量</th><th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="b in books" :key="b.id">
            <td class="bkm-strong">{{ b.title }}</td>
            <td>{{ b.author || "—" }}</td>
            <td class="bkm-mono">{{ b.isbn || "—" }}</td>
            <td>{{ b.category || "—" }}</td>
            <td>{{ b.location || "—" }}</td>
            <td class="num"><span :class="{ out: b.available <= 0 }">{{ b.available }}</span> / {{ b.total }}</td>
            <td class="bkm-row-actions">
              <button class="btn btn-sm" @click="openEdit(b)">编辑</button>
              <button class="btn btn-sm danger" @click="doDelete(b)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="!loading && totalPages > 1" class="bkm-pager">
        <button class="btn btn-sm" :disabled="page <= 1" @click="goPage(page - 1)">上一页</button>
        <span class="bkm-pageinfo">第 {{ page }} / {{ totalPages }} 页 · 共 {{ total }} 册</span>
        <button class="btn btn-sm" :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</button>
      </div>
    </section>

    <!-- All borrow records -->
    <section v-else>
      <div v-if="loadingBorrows" class="bkm-empty">加载中…</div>
      <div v-else-if="borrows.length === 0" class="bkm-empty">暂无借阅记录</div>
      <table v-else class="bkm-table">
        <thead>
          <tr>
            <th>书名</th><th>ISBN</th><th>借阅人</th><th>借出时间</th><th>应还时间</th><th>状态</th><th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in borrows" :key="r.id">
            <td class="bkm-strong">{{ r.book_title }}</td>
            <td class="bkm-mono">{{ r.book_isbn || "—" }}</td>
            <td class="bkm-mono">{{ shortId(r.member_id) }}</td>
            <td>{{ fmt(r.borrowed_at) }}</td>
            <td :class="{ overdue: isOverdue(r) }">{{ fmt(r.due_at) }}</td>
            <td><span class="bkm-status" :class="statusClass(r)">{{ statusLabel(r) }}</span></td>
            <td>
              <button v-if="!r.returned_at" class="btn btn-sm" :disabled="returning === r.id" @click="doReturn(r)">归还</button>
              <span v-else class="bkm-muted">{{ fmt(r.returned_at) }} 归还</span>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <!-- Create / edit modal -->
    <div v-if="modal.open" class="bkm-overlay" @click.self="modal.open = false">
      <div class="bkm-modal">
        <h3>{{ modal.id ? "编辑图书" : "新增图书" }}</h3>
        <div class="bkm-form">
          <label class="bkm-field full">
            <span>书名 *</span>
            <input v-model="modal.title" autofocus />
          </label>
          <label class="bkm-field">
            <span>作者</span>
            <input v-model="modal.author" />
          </label>
          <label class="bkm-field">
            <span>ISBN</span>
            <input v-model="modal.isbn" />
          </label>
          <label class="bkm-field">
            <span>出版社</span>
            <input v-model="modal.publisher" />
          </label>
          <label class="bkm-field">
            <span>分类</span>
            <input v-model="modal.category" />
          </label>
          <label class="bkm-field">
            <span>馆藏位置</span>
            <input v-model="modal.location" placeholder="如 A区-3排" />
          </label>
          <label class="bkm-field">
            <span>馆藏总量</span>
            <input v-model.number="modal.total" type="number" min="0" />
          </label>
          <label class="bkm-field full">
            <span>封面图 URL</span>
            <input v-model="modal.cover_url" placeholder="https://…" />
          </label>
        </div>
        <div class="bkm-modal-foot">
          <button class="btn btn-ghost" @click="modal.open = false">取消</button>
          <button class="btn btn-primary" :disabled="saving" @click="save">{{ modal.id ? "保存" : "创建" }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import * as api from "../api/books";
import type { Book, Borrow, BorrowStatus } from "../api/books";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

const notify = useNotification();
const { confirm } = useConfirm();

const tab = ref<"books" | "borrows">("books");

// ---- books ----
const books = ref<Book[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 20;
const search = ref("");
const loading = ref(false);
const saving = ref(false);

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)));

async function loadBooks() {
  loading.value = true;
  try {
    const r = await api.listBooks({ search: search.value.trim(), page: page.value, page_size: pageSize });
    books.value = r.books;
    total.value = r.total;
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载图书失败");
  } finally {
    loading.value = false;
  }
}
function doSearch() { page.value = 1; loadBooks(); }
function goPage(p: number) { page.value = p; loadBooks(); }

// ---- modal ----
const modal = reactive({
  open: false, id: "", isbn: "", title: "", author: "", publisher: "",
  category: "", total: 1, cover_url: "", location: "",
});
function reset() {
  modal.id = ""; modal.isbn = ""; modal.title = ""; modal.author = ""; modal.publisher = "";
  modal.category = ""; modal.total = 1; modal.cover_url = ""; modal.location = "";
}
function openCreate() { reset(); modal.open = true; }
function openEdit(b: Book) {
  modal.id = b.id; modal.isbn = b.isbn; modal.title = b.title; modal.author = b.author;
  modal.publisher = b.publisher; modal.category = b.category; modal.total = b.total;
  modal.cover_url = b.cover_url; modal.location = b.location;
  modal.open = true;
}
async function save() {
  if (!modal.title.trim()) { notify.warning("请输入书名"); return; }
  saving.value = true;
  const payload = {
    isbn: modal.isbn, title: modal.title.trim(), author: modal.author, publisher: modal.publisher,
    category: modal.category, total: Number(modal.total) || 0, cover_url: modal.cover_url, location: modal.location,
  };
  try {
    if (modal.id) await api.updateBook(modal.id, payload);
    else await api.createBook(payload);
    modal.open = false;
    await loadBooks();
    notify.success("已保存");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "保存失败");
  } finally {
    saving.value = false;
  }
}
async function doDelete(b: Book) {
  if (!(await confirm({ title: "删除图书", message: `确认删除《${b.title}》及其借阅记录？`, danger: true }))) return;
  try {
    await api.deleteBook(b.id);
    await loadBooks();
    notify.success("已删除");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}

// ---- borrows (all) ----
const borrows = ref<Borrow[]>([]);
const loadingBorrows = ref(false);
const borrowFilter = ref<"" | BorrowStatus>("");
const returning = ref<string | null>(null);

async function loadBorrows() {
  loadingBorrows.value = true;
  try {
    borrows.value = await api.listBorrows("all", borrowFilter.value || undefined);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载借阅记录失败");
  } finally {
    loadingBorrows.value = false;
  }
}
function switchToBorrows() { tab.value = "borrows"; loadBorrows(); }

async function doReturn(r: Borrow) {
  if (!(await confirm({ title: "归还图书", message: `确认将《${r.book_title}》标记为已归还？` }))) return;
  returning.value = r.id;
  try {
    await api.returnBorrow(r.id);
    notify.success("已归还");
    await loadBorrows();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "归还失败");
  } finally {
    returning.value = null;
  }
}

function fmt(iso?: string | null) {
  if (!iso) return "—";
  const d = new Date(iso);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
function shortId(id: string) { return id ? id.slice(0, 8) : "—"; }
function isOverdue(r: Borrow) { return !r.returned_at && new Date(r.due_at) < new Date(); }
function statusLabel(r: Borrow) {
  if (r.returned_at) return "已归还";
  if (isOverdue(r)) return "已逾期";
  return "借阅中";
}
function statusClass(r: Borrow) {
  if (r.returned_at) return "returned";
  if (isOverdue(r)) return "overdue";
  return "borrowed";
}

onMounted(loadBooks);
</script>

<style scoped>
.bkm-app { padding: 4px 2px; }
.bkm-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 18px; flex-wrap: wrap; }
.bkm-tabs { display: flex; gap: 4px; }
.bkm-tab { border: 0; background: transparent; padding: 8px 14px; border-radius: 8px; cursor: pointer; font-size: 14px; color: var(--text-2); font-weight: 600; }
.bkm-tab:hover { background: var(--surface-2); }
.bkm-tab.active { background: var(--primary-soft); color: var(--primary); }
.bkm-actions { display: flex; gap: 8px; align-items: center; }
.bkm-input { border: 1px solid var(--border); border-radius: 8px; padding: 7px 11px; font-size: 13px; background: var(--surface); color: var(--text); outline: none; }
.bkm-input:focus { border-color: var(--primary); }

.bkm-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.bkm-table th, .bkm-table td { text-align: left; padding: 10px 12px; border-bottom: 1px solid var(--border); }
.bkm-table th { font-size: 12px; color: var(--text-3); font-weight: 600; }
.bkm-table th.num, .bkm-table td.num { text-align: right; }
.bkm-strong { font-weight: 600; color: var(--text); }
.bkm-mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; color: var(--text-2); }
.bkm-muted { font-size: 11px; color: var(--text-4); }
.num .out { color: var(--danger); }
.overdue { color: var(--danger); font-weight: 600; }
.bkm-row-actions { display: flex; gap: 6px; }

.bkm-status { font-size: 11px; padding: 2px 8px; border-radius: 999px; }
.bkm-status.borrowed { color: var(--primary); background: var(--primary-soft); }
.bkm-status.returned { color: var(--text-3); background: var(--surface-2); }
.bkm-status.overdue { color: var(--danger); background: var(--danger-soft); }

.bkm-pager { display: flex; align-items: center; justify-content: center; gap: 14px; margin-top: 20px; }
.bkm-pageinfo { font-size: 12px; color: var(--text-3); }
.bkm-empty { color: var(--text-3); text-align: center; padding: 50px 0; font-size: 13px; }

.bkm-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.bkm-modal { background: var(--surface); border-radius: 14px; padding: 24px; width: min(560px, 94vw); box-shadow: var(--sh-4); }
.bkm-modal h3 { font-size: 16px; font-weight: 600; margin: 0 0 16px; }
.bkm-form { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.bkm-field { display: flex; flex-direction: column; gap: 5px; }
.bkm-field.full { grid-column: 1 / -1; }
.bkm-field span { font-size: 12px; color: var(--text-3); }
.bkm-field input {
  border: 1px solid var(--border); border-radius: 8px; padding: 7px 10px; font-size: 13px;
  background: var(--surface); color: var(--text); outline: none;
}
.bkm-field input:focus { border-color: var(--primary); }
.bkm-modal-foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 20px; }

.btn-sm { padding: 4px 12px; font-size: 12px; }
.btn.danger { color: var(--danger); }
</style>
