<template>
  <section class="me-page">
    <PageHeader title="我的工作区" sub="你所属的所有组织 · 点击卡片切换" />

    <div v-if="auth.tenants.length === 0" class="empty">
      <div class="empty-title">你尚未加入任何组织</div>
      <div class="empty-sub">请联系管理员添加</div>
    </div>

    <div v-else class="tenant-grid">
      <article
        v-for="t in auth.tenants"
        :key="t.id"
        class="tenant-card"
        :class="{ active: t.id === auth.tenant?.id }"
        @click="switchTo(t.id)"
      >
        <div class="t-logo" :style="{ background: colorFor(t.name) }">{{ t.name[0] }}</div>
        <div class="t-name">{{ t.name }}</div>
        <div class="t-slug"><code>{{ t.slug }}</code></div>
        <div v-if="t.id === auth.tenant?.id" class="badge-current">当前</div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { PageHeader } from "@/shell/components";
import { useAuthStore } from "@/shell/auth/auth.store";
import { useRouter } from "vue-router";

const auth = useAuthStore();
const router = useRouter();

async function switchTo(id: string) {
  if (id === auth.tenant?.id) return;
  await auth.switchTenant(id);
  router.push("/");
}

function colorFor(name: string) {
  const seed = (name || "?").split("").reduce((s, c) => s + c.charCodeAt(0), 0);
  const palette = [
    "linear-gradient(135deg,#1e5fd9,#4a85ee)",
    "linear-gradient(135deg,#7c4ddb,#5a2db5)",
    "linear-gradient(135deg,#0fa8a3,#0a7e7a)",
    "linear-gradient(135deg,#e8920e,#b86d05)",
    "linear-gradient(135deg,#1aa971,#0e7b51)",
  ];
  return palette[seed % palette.length];
}
</script>

<style scoped>
.me-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.empty { padding: 40px; text-align: center; color: var(--text-3); background: var(--surface); border-radius: 14px; }
.empty-title { font-size: 14px; font-weight: 600; color: var(--text-2); }
.empty-sub { font-size: 12.5px; margin-top: 4px; }

.tenant-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 14px;
}
.tenant-card {
  position: relative;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 22px 18px;
  text-align: center;
  cursor: pointer;
  transition: all 0.15s;
}
.tenant-card:hover {
  transform: translateY(-2px);
  border-color: var(--border-strong);
  box-shadow: var(--sh-2);
}
.tenant-card.active {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-soft);
}
.t-logo {
  width: 48px; height: 48px;
  border-radius: 10px;
  color: #fff;
  font-size: 20px;
  font-weight: 700;
  display: grid; place-items: center;
  margin: 0 auto 10px;
  box-shadow: 0 3px 8px rgba(13,27,46,.12);
}
.t-name { font-size: 14px; font-weight: 600; color: var(--text); }
.t-slug { font-size: 11.5px; color: var(--text-3); margin-top: 4px; }
.t-slug code {
  font-family: var(--ff-mono);
  background: var(--surface-2);
  padding: 1px 6px;
  border-radius: 3px;
}
.badge-current {
  position: absolute;
  top: 10px; right: 10px;
  background: var(--primary);
  color: #fff;
  font-size: 10.5px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 999px;
}
</style>
