<template>
  <section class="workspace">
    <h1>工作台</h1>
    <p class="muted">欢迎使用 IOP. M1 仅基座可用; M2 起加租户与登录, M4 加 OKR 业务模块.</p>

    <div class="card">
      <h2>系统状态</h2>
      <ul>
        <li><strong>版本:</strong> {{ version || '加载中…' }}</li>
        <li><strong>API 基址:</strong> <code>{{ apiBase }}</code></li>
        <li><strong>后端可达性:</strong>
          <span :class="['status', readyz.live ? 'ok' : 'err']">
            {{ readyz.live ? '✓ live' : '✗ unreachable' }}
          </span>
          <span :class="['status', readyz.ready ? 'ok' : 'err']" v-if="readyz.live">
            {{ readyz.ready ? '✓ ready' : '✗ not ready' }}
          </span>
        </li>
      </ul>
    </div>

    <div class="card" v-if="dictItems.length">
      <h2>字典: plan_level (来自后端 services/dictionary)</h2>
      <ul>
        <li v-for="item in dictItems" :key="item.code">
          <code>{{ item.code }}</code> — {{ item.name }}
        </li>
      </ul>
    </div>

    <div class="card" v-if="lastError">
      <h2 class="err">最近一次错误响应 (验证 envelope 格式)</h2>
      <pre>{{ JSON.stringify(lastError, null, 2) }}</pre>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { client } from "@/api/client";

interface DictItem {
  type_code: string;
  code: string;
  name: string;
  sort_order: number;
  active: boolean;
}

const version = ref<string>("");
const apiBase = import.meta.env.MODE === "development" ? "/api (proxy → :8080)" : "/api";
const readyz = ref<{ live: boolean; ready: boolean }>({ live: false, ready: false });
const dictItems = ref<DictItem[]>([]);
const lastError = ref<unknown>(null);

onMounted(async () => {
  // version — root endpoint, bypass /api prefix via vite proxy
  try {
    const res = await fetch("/version");
    const j = await res.json();
    version.value = (j?.version as string) ?? "unknown";
  } catch {
    version.value = "(后端不可达)";
  }
  // readyz (note: not under /api, accessed directly via proxy)
  try {
    const res = await fetch("/readyz");
    readyz.value = await res.json();
  } catch {
    readyz.value = { live: false, ready: false };
  }
  // dictionary (real API call through envelope)
  try {
    const res = await client.get("/dict/plan_level");
    dictItems.value = res.data?.data?.items ?? [];
  } catch (e: any) {
    dictItems.value = [];
  }
  // probe error envelope shape
  try {
    await client.get("/dict/__doesnotexist__");
  } catch (e: any) {
    lastError.value = e.response?.data ?? { message: String(e) };
  }
});
</script>

<style scoped>
.workspace {
  max-width: 800px;
  margin: 0 auto;
}
h1 { margin-bottom: var(--space-2); }
.muted { color: var(--color-text-muted); margin-bottom: var(--space-6); }
.card {
  margin-top: var(--space-6);
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}
.card h2 { font-size: 16px; margin-bottom: var(--space-3); }
.card ul { list-style: none; padding-left: 0; }
.card li { padding: var(--space-1) 0; }
code {
  background: var(--color-bg);
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 13px;
}
pre {
  background: var(--color-bg);
  padding: var(--space-3);
  border-radius: var(--radius);
  font-size: 12px;
  overflow-x: auto;
}
.status { padding: 2px 8px; border-radius: 4px; margin-left: var(--space-2); }
.status.ok { background: rgba(0, 168, 112, 0.1); color: var(--color-success); }
.status.err { background: rgba(213, 73, 65, 0.1); color: var(--color-danger); }
.err { color: var(--color-danger); }
</style>
