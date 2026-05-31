<template>
  <div class="books-app">
    <header class="bk-head">
      <div class="bk-tabs">
        <button class="bk-tab" :class="{ active: tab === 'catalog' }" @click="tab = 'catalog'">图书目录</button>
        <button class="bk-tab" :class="{ active: tab === 'mine' }" @click="switchToMine">我的借阅</button>
      </div>
      <div v-if="tab === 'catalog'" class="bk-search">
        <input
          v-model="search"
          class="bk-search-input"
          placeholder="搜索书名 / 作者 / ISBN / 出版社"
          @keyup.enter="doSearch"
        />
        <button class="btn btn-primary" @click="doSearch">搜索</button>
      </div>
    </header>

    <!-- Catalog -->
    <section v-if="tab === 'catalog'">
      <div v-if="loading" class="bk-empty">加载中…</div>
      <div v-else-if="books.length === 0" class="bk-empty">
        <div class="bk-empty-emoji">📚</div>
        <div>没有找到图书</div>
      </div>

      <ul v-else class="bk-grid">
        <li v-for="b in books" :key="b.id" class="bk-card">
          <div class="bk-cover" :style="b.cover_url ? { backgroundImage: `url(${b.cover_url})` } : {}">
            <span v-if="!b.cover_url" class="bk-cover-ph">{{ b.title.slice(0, 1) }}</span>
          </div>
          <div class="bk-info">
            <div class="bk-title" :title="b.title">{{ b.title }}</div>
            <div class="bk-author">{{ b.author || "佚名" }}</div>
            <div class="bk-meta">
              <span v-if="b.category" class="bk-pill">{{ b.category }}</span>
              <span v-if="b.location" class="bk-loc">📍 {{ b.location }}</span>
            </div>
            <div class="bk-foot">
              <span class="bk-avail" :class="{ out: b.available <= 0 }">
                可借 {{ b.available }} / {{ b.total }}
              </span>
              <button
                class="btn btn-sm btn-primary"
                :disabled="b.available <= 0 || borrowing === b.id"
                @click="doBorrow(b)"
              >
                {{ b.available <= 0 ? "已借完" : (borrowing === b.id ? "借阅中…" : "借阅") }}
              </button>
            </div>
          </div>
        </li>
      </ul>

      <div v-if="!loading && totalPages > 1" class="bk-pager">
        <button class="btn btn-sm" :disabled="page <= 1" @click="goPage(page - 1)">上一页</button>
        <span class="bk-pageinfo">第 {{ page }} / {{ totalPages }} 页 · 共 {{ total }} 册</span>
        <button class="btn btn-sm" :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</button>
      </div>
    </section>

    <!-- My borrows -->
    <section v-else>
      <div v-if="loadingBorrows" class="bk-empty">加载中…</div>
      <div v-else-if="myBorrows.length === 0" class="bk-empty">
        <div class="bk-empty-emoji">🗂</div>
        <div>你还没有借阅记录</div>
      </div>
      <table v-else class="bk-table">
        <thead>
          <tr>
            <th>书名</th><th>作者</th><th>借出时间</th><th>应还时间</th><th>状态</th><th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in myBorrows" :key="r.id">
            <td class="bk-strong">{{ r.book_title }}</td>
            <td>{{ r.book_author || "—" }}</td>
            <td>{{ fmt(r.borrowed_at) }}</td>
            <td :class="{ overdue: isOverdue(r) }">{{ fmt(r.due_at) }}</td>
            <td><span class="bk-status" :class="statusClass(r)">{{ statusLabel(r) }}</span></td>
            <td>
              <button
                v-if="!r.returned_at"
                class="btn btn-sm"
                :disabled="returning === r.id"
                @click="doReturn(r)"
              >归还</button>
              <span v-else class="bk-returned">已于 {{ fmt(r.returned_at) }} 归还</span>
            </td>
          </tr>
        </tbody>
      </table>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import * as api from "../api/books";
import type { Book, Borrow } from "../api/books";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

const notify = useNotification();
const { confirm } = useConfirm();

const tab = ref<"catalog" | "mine">("catalog");

// ---- catalog ----
const books = ref<Book[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 12;
const search = ref("");
const loading = ref(false);
const borrowing = ref<string | null>(null);

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

async function doBorrow(b: Book) {
  borrowing.value = b.id;
  try {
    await api.borrowBook(b.id);
    notify.success(`已借阅《${b.title}》`);
    await loadBooks();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "借阅失败");
  } finally {
    borrowing.value = null;
  }
}

// ---- my borrows ----
const myBorrows = ref<Borrow[]>([]);
const loadingBorrows = ref(false);
const returning = ref<string | null>(null);

async function loadMine() {
  loadingBorrows.value = true;
  try {
    myBorrows.value = await api.listBorrows("mine");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载借阅记录失败");
  } finally {
    loadingBorrows.value = false;
  }
}
function switchToMine() { tab.value = "mine"; loadMine(); }

async function doReturn(r: Borrow) {
  if (!(await confirm({ title: "归还图书", message: `确认归还《${r.book_title}》？` }))) return;
  returning.value = r.id;
  try {
    await api.returnBorrow(r.id);
    notify.success("已归还");
    await loadMine();
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
function isOverdue(r: Borrow) {
  return !r.returned_at && new Date(r.due_at) < new Date();
}
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
.books-app { padding: 4px 2px; }
.bk-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 18px; flex-wrap: wrap; }
.bk-tabs { display: flex; gap: 4px; }
.bk-tab {
  border: 0; background: transparent; padding: 8px 14px; border-radius: 8px;
  cursor: pointer; font-size: 14px; color: var(--text-2); font-weight: 600;
}
.bk-tab:hover { background: var(--surface-2); }
.bk-tab.active { background: var(--primary-soft); color: var(--primary); }
.bk-search { display: flex; gap: 8px; }
.bk-search-input {
  width: 320px; max-width: 60vw; border: 1px solid var(--border); border-radius: 8px;
  padding: 8px 12px; font-size: 13px; background: var(--surface); color: var(--text); outline: none;
}
.bk-search-input:focus { border-color: var(--primary); }

.bk-grid {
  list-style: none; margin: 0; padding: 0;
  display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 14px;
}
.bk-card {
  display: flex; gap: 12px; background: var(--surface); border: 1px solid var(--border);
  border-radius: 12px; padding: 12px; transition: box-shadow .15s, border-color .15s;
}
.bk-card:hover { border-color: var(--primary); box-shadow: var(--sh-2); }
.bk-cover {
  width: 64px; height: 90px; border-radius: 6px; flex-shrink: 0;
  background: linear-gradient(135deg, var(--primary-soft), var(--surface-2));
  background-size: cover; background-position: center;
  display: grid; place-items: center;
}
.bk-cover-ph { font-size: 28px; font-weight: 700; color: var(--primary); }
.bk-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 4px; }
.bk-title { font-size: 14px; font-weight: 600; color: var(--text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.bk-author { font-size: 12px; color: var(--text-3); }
.bk-meta { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-top: 2px; }
.bk-pill { font-size: 11px; color: var(--primary); background: var(--primary-soft); padding: 1px 7px; border-radius: 4px; }
.bk-loc { font-size: 11px; color: var(--text-4); }
.bk-foot { margin-top: auto; display: flex; align-items: center; justify-content: space-between; padding-top: 6px; }
.bk-avail { font-size: 12px; color: var(--success); font-weight: 600; }
.bk-avail.out { color: var(--text-4); }

.bk-pager { display: flex; align-items: center; justify-content: center; gap: 14px; margin-top: 20px; }
.bk-pageinfo { font-size: 12px; color: var(--text-3); }

.bk-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.bk-table th, .bk-table td { text-align: left; padding: 10px 12px; border-bottom: 1px solid var(--border); }
.bk-table th { font-size: 12px; color: var(--text-3); font-weight: 600; }
.bk-strong { font-weight: 600; color: var(--text); }
.overdue { color: var(--danger); font-weight: 600; }
.bk-status { font-size: 11px; padding: 2px 8px; border-radius: 999px; }
.bk-status.borrowed { color: var(--primary); background: var(--primary-soft); }
.bk-status.returned { color: var(--text-3); background: var(--surface-2); }
.bk-status.overdue { color: var(--danger); background: var(--danger-soft); }
.bk-returned { font-size: 11px; color: var(--text-4); }

.bk-empty { color: var(--text-3); text-align: center; padding: 50px 0; font-size: 13px; }
.bk-empty-emoji { font-size: 34px; margin-bottom: 8px; }

.btn-sm { padding: 4px 12px; font-size: 12px; }
</style>
