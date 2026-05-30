<template>
  <div class="news-feed">
    <!-- Left: category rail -->
    <aside class="nf-side">
      <div class="nf-side-head">栏目</div>
      <button
        class="nf-cat" :class="{ active: selCat === '' }"
        @click="selectCat('')"
      >
        <span class="nf-cat-label">全部资讯</span>
      </button>
      <button
        v-for="c in categories" :key="c.id"
        class="nf-cat" :class="{ active: selCat === c.id }"
        @click="selectCat(c.id)"
      >
        <span class="nf-cat-label">{{ c.name }}</span>
        <span v-if="c.article_count" class="nf-cat-count">{{ c.article_count }}</span>
      </button>
    </aside>

    <!-- Center: article list -->
    <main class="nf-main" v-if="!current">
      <header class="nf-main-head">
        <h2>{{ currentCatName }}</h2>
        <span class="nf-total" v-if="total">共 {{ total }} 篇</span>
      </header>

      <div v-if="loading" class="nf-loading">加载中…</div>
      <div v-else-if="articles.length === 0" class="nf-empty">
        <div class="nf-empty-emoji">📰</div>
        <div>该栏目暂无已发布资讯</div>
      </div>

      <ul v-else class="nf-list">
        <li v-for="a in articles" :key="a.id" class="nf-item" @click="openArticle(a.id)">
          <div v-if="a.cover_url" class="nf-cover" :style="{ backgroundImage: `url(${a.cover_url})` }"></div>
          <div class="nf-item-body">
            <h3 class="nf-item-title">{{ a.title }}</h3>
            <p class="nf-item-summary">{{ a.summary || stripPreview(a.content) }}</p>
            <div class="nf-item-meta">
              <span v-if="a.category_name" class="nf-tag">{{ a.category_name }}</span>
              <span v-if="a.author">{{ a.author }}</span>
              <span>{{ fmtDate(a.published_at) }}</span>
              <span class="nf-views">👁 {{ a.views }}</span>
            </div>
          </div>
        </li>
      </ul>

      <div v-if="totalPages > 1" class="nf-pager">
        <button :disabled="page <= 1" @click="goPage(page - 1)">上一页</button>
        <span>{{ page }} / {{ totalPages }}</span>
        <button :disabled="page >= totalPages" @click="goPage(page + 1)">下一页</button>
      </div>
    </main>

    <!-- Detail reader -->
    <article class="nf-reader" v-else>
      <button class="nf-back" @click="current = null">← 返回列表</button>
      <div v-if="loadingDetail" class="nf-loading">加载中…</div>
      <template v-else>
        <h1 class="nf-reader-title">{{ current.title }}</h1>
        <div class="nf-reader-meta">
          <span v-if="current.category_name" class="nf-tag">{{ current.category_name }}</span>
          <span v-if="current.author">{{ current.author }}</span>
          <span>{{ fmtDate(current.published_at) }}</span>
          <span class="nf-views">👁 {{ current.views }}</span>
        </div>
        <img v-if="current.cover_url" :src="current.cover_url" class="nf-reader-cover" alt="" />
        <p v-if="current.summary" class="nf-reader-summary">{{ current.summary }}</p>
        <div class="nf-reader-content">{{ current.content }}</div>
      </template>
    </article>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useNotification } from "@/shell/notify";
import {
  listCategories,
  listFeed,
  getArticle,
  type NewsCategory,
  type NewsArticle,
} from "../api";

const notify = useNotification();

const categories = ref<NewsCategory[]>([]);
const articles = ref<NewsArticle[]>([]);
const selCat = ref("");
const page = ref(1);
const pageSize = 10;
const total = ref(0);
const loading = ref(false);

const current = ref<NewsArticle | null>(null);
const loadingDetail = ref(false);

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)));
const currentCatName = computed(() => {
  if (!selCat.value) return "全部资讯";
  return categories.value.find((c) => c.id === selCat.value)?.name ?? "资讯";
});

function fmtDate(s?: string | null): string {
  if (!s) return "";
  const d = new Date(s);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
function stripPreview(content: string): string {
  return (content || "").replace(/\s+/g, " ").slice(0, 80);
}

async function loadFeed() {
  loading.value = true;
  try {
    const r = await listFeed(selCat.value, page.value, pageSize);
    articles.value = r.items;
    total.value = r.total;
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载资讯失败");
  } finally {
    loading.value = false;
  }
}

function selectCat(id: string) {
  selCat.value = id;
  page.value = 1;
  current.value = null;
  loadFeed();
}
function goPage(p: number) {
  page.value = p;
  loadFeed();
}

async function openArticle(id: string) {
  loadingDetail.value = true;
  current.value = { id } as NewsArticle;
  try {
    current.value = await getArticle(id, true); // track=true bumps views
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载文章失败");
    current.value = null;
  } finally {
    loadingDetail.value = false;
  }
}

onMounted(async () => {
  try {
    categories.value = await listCategories();
  } catch {
    /* read perm only — categories optional for layout */
  }
  loadFeed();
});
</script>

<style scoped>
.news-feed {
  display: flex;
  height: 100%;
  gap: 16px;
}
.nf-side {
  width: 200px;
  flex-shrink: 0;
  border-right: 1px solid var(--border, #e5e7eb);
  padding: 12px 8px;
  overflow-y: auto;
}
.nf-side-head {
  font-size: 12px;
  color: var(--text-4, #9ca3af);
  padding: 4px 10px 8px;
  letter-spacing: 0.05em;
}
.nf-cat {
  display: flex;
  align-items: center;
  width: 100%;
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
.nf-cat:hover { background: var(--bg-2, #f3f4f6); }
.nf-cat.active { background: var(--bg-active, #e0e7ff); color: var(--cat-collab, #4f46e5); font-weight: 600; }
.nf-cat-label { flex: 1; }
.nf-cat-count { font-size: 12px; color: var(--text-4, #9ca3af); }

.nf-main { flex: 1; overflow-y: auto; padding: 12px 16px; }
.nf-main-head { display: flex; align-items: baseline; gap: 12px; margin-bottom: 12px; }
.nf-main-head h2 { margin: 0; font-size: 20px; }
.nf-total { font-size: 13px; color: var(--text-4, #9ca3af); }

.nf-list { list-style: none; margin: 0; padding: 0; }
.nf-item {
  display: flex;
  gap: 14px;
  padding: 14px 8px;
  border-bottom: 1px solid var(--border, #eee);
  cursor: pointer;
  border-radius: 8px;
}
.nf-item:hover { background: var(--bg-2, #f9fafb); }
.nf-cover {
  width: 140px;
  height: 90px;
  flex-shrink: 0;
  border-radius: 8px;
  background-size: cover;
  background-position: center;
  background-color: var(--bg-2, #f3f4f6);
}
.nf-item-body { flex: 1; min-width: 0; }
.nf-item-title { margin: 0 0 6px; font-size: 16px; font-weight: 600; }
.nf-item-summary {
  margin: 0 0 8px;
  color: var(--text-3, #6b7280);
  font-size: 13px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.nf-item-meta { display: flex; gap: 12px; font-size: 12px; color: var(--text-4, #9ca3af); align-items: center; }
.nf-tag {
  padding: 1px 8px;
  background: var(--bg-active, #eef2ff);
  color: var(--cat-collab, #4f46e5);
  border-radius: 10px;
}
.nf-views { margin-left: auto; }

.nf-pager { display: flex; gap: 12px; align-items: center; justify-content: center; padding: 16px; }
.nf-pager button {
  padding: 6px 14px;
  border: 1px solid var(--border, #d1d5db);
  background: #fff;
  border-radius: 8px;
  cursor: pointer;
}
.nf-pager button:disabled { opacity: 0.45; cursor: not-allowed; }

.nf-loading, .nf-empty { padding: 48px; text-align: center; color: var(--text-4, #9ca3af); }
.nf-empty-emoji { font-size: 36px; margin-bottom: 8px; }

.nf-reader { flex: 1; overflow-y: auto; padding: 16px 32px; max-width: 860px; }
.nf-back {
  border: none;
  background: transparent;
  color: var(--cat-collab, #4f46e5);
  cursor: pointer;
  font-size: 14px;
  margin-bottom: 16px;
  padding: 4px 0;
}
.nf-reader-title { font-size: 28px; margin: 0 0 12px; line-height: 1.3; }
.nf-reader-meta { display: flex; gap: 14px; align-items: center; font-size: 13px; color: var(--text-4, #9ca3af); margin-bottom: 16px; }
.nf-reader-cover { width: 100%; border-radius: 12px; margin-bottom: 16px; }
.nf-reader-summary {
  font-size: 15px;
  color: var(--text-2, #374151);
  background: var(--bg-2, #f9fafb);
  border-left: 3px solid var(--cat-collab, #4f46e5);
  padding: 12px 16px;
  border-radius: 0 8px 8px 0;
  margin-bottom: 20px;
}
.nf-reader-content { font-size: 16px; line-height: 1.9; color: var(--text-1, #1f2937); white-space: pre-wrap; }
</style>
