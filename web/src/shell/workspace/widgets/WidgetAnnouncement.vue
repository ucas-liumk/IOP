<template>
  <WidgetCard
    title="平台公告"
    :icon="iconPath"
    :source="{ code: 'iop', name: '平台', color: 'var(--warning)' }"
    :config-mode="configMode"
  >
    <ul class="news">
      <li v-for="n in NEWS" :key="n.title" class="news-row">
        <span class="tag" :style="{ background: n.tagBg, color: n.tagFg }">{{ n.tag }}</span>
        <span class="title">{{ n.title }}</span>
        <span class="date">{{ n.date }}</span>
      </li>
    </ul>
  </WidgetCard>
</template>

<script setup lang="ts">
import WidgetCard from "./WidgetCard.vue";

defineProps<{ configMode?: boolean }>();

const iconPath = "M18 11v-1l-5-5H7c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h10c1.1 0 2-.9 2-2v-9.18A4 4 0 0 1 18 11zM12 19l-4-4h2.5v-4h3v4H16l-4 4z";

// Curated platform release-notes feed. Replace with /platform/announcements
// once a backend exists.
const NEWS = [
  { tag: "新功能", tagBg: "var(--info-soft)", tagFg: "var(--info)",
    title: "工作台支持自定义组件", date: "2026-05-30" },
  { tag: "新功能", tagBg: "var(--info-soft)", tagFg: "var(--info)",
    title: "应用中心上线", date: "2026-05-26" },
  { tag: "优化", tagBg: "var(--warning-soft)", tagFg: "var(--warning)",
    title: "OKR 周报模板可定制", date: "2026-05-22" },
  { tag: "公告", tagBg: "var(--purple-soft)", tagFg: "var(--purple)",
    title: "本周日凌晨数据库维护 30 分钟", date: "2026-05-20" },
];
</script>

<style scoped>
.news { list-style: none; margin: 0; padding: 0; }
.news-row {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 0;
  font-size: 13px;
  color: var(--text);
  border-bottom: 1px dashed var(--border);
}
.news-row:last-child { border-bottom: 0; }
.tag {
  font-size: 10.5px;
  font-weight: 600;
  padding: 2px 7px;
  border-radius: 6px;
  white-space: nowrap;
  flex-shrink: 0;
}
.title {
  flex: 1; min-width: 0;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.date {
  font-size: 11px;
  color: var(--text-3);
  font-family: var(--ff-mono);
  flex-shrink: 0;
}
</style>
