<template>
  <section>
    <header class="page-head">
      <h1>部门管理</h1>
      <p class="sub">从成员档案 (member.department) 派生，编辑某成员的部门字段即可调整归属</p>
    </header>

    <div class="grid">
      <article v-for="d in depts" :key="d.name" class="dept-card">
        <div class="dept-head">
          <div class="dept-icon" :style="{ background: colorFor(d.name) }">
            {{ d.name[0] }}
          </div>
          <div>
            <div class="dept-name">{{ d.name }}</div>
            <div class="dept-meta">{{ d.members.length }} 人</div>
          </div>
        </div>
        <ul class="member-list">
          <li v-for="m in d.members" :key="m.member_id">
            <span class="avatar" :style="{ background: colorFor(m.display_name) }">{{ m.display_name[0] }}</span>
            <div>
              <div class="m-name">{{ m.display_name }}</div>
              <div class="m-title">{{ m.title || '—' }}</div>
            </div>
          </li>
        </ul>
        <router-link :to="`/admin/members?dept=${encodeURIComponent(d.name)}`" class="manage-link">管理 →</router-link>
      </article>
      <div v-if="depts.length === 0" class="empty">尚无部门 · 编辑成员档案 → 填写部门后会自动出现</div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { listMembers, type Member } from "../api/admin";

const members = ref<Member[]>([]);

const depts = computed(() => {
  const map = new Map<string, Member[]>();
  for (const m of members.value) {
    const key = m.department || "未分配";
    if (!map.has(key)) map.set(key, []);
    map.get(key)!.push(m);
  }
  return Array.from(map.entries()).map(([name, ms]) => ({ name, members: ms }));
});

onMounted(async () => { members.value = await listMembers(); });

function colorFor(name: string) {
  const seed = name.split("").reduce((s, c) => s + c.charCodeAt(0), 0);
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
.page-head { margin-bottom: 20px; }
.page-head h1 { font-size: 22px; font-weight: 700; }
.page-head .sub { font-size: 13px; color: var(--text-3); margin-top: 4px; }

.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 14px; }

.dept-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dept-head { display: flex; gap: 12px; align-items: center; }
.dept-icon {
  width: 40px; height: 40px;
  border-radius: 10px;
  display: grid; place-items: center;
  color: white; font-weight: 700; font-size: 16px;
}
.dept-name { font-size: 15px; font-weight: 600; }
.dept-meta { font-size: 12px; color: var(--text-3); }

.member-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 5px;
  border-top: 1px solid var(--border-soft);
  padding-top: 10px;
}
.member-list li {
  display: flex; gap: 8px; align-items: center;
}
.avatar {
  width: 26px; height: 26px;
  border-radius: 50%;
  color: white; font-weight: 600; font-size: 11px;
  display: grid; place-items: center;
  flex-shrink: 0;
}
.m-name { font-size: 13px; font-weight: 500; }
.m-title { font-size: 11px; color: var(--text-3); }

.manage-link {
  display: block;
  text-align: right;
  font-size: 12.5px;
  color: var(--primary);
  margin-top: auto;
  padding-top: 6px;
  border-top: 1px solid var(--border-soft);
}

.empty {
  grid-column: 1 / -1;
  text-align: center;
  padding: 60px 0;
  color: var(--text-4);
  font-size: 13px;
}
</style>
