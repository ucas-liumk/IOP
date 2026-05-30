<template>
  <Teleport to="body">
    <div v-if="open" class="wg-overlay" @click.self="$emit('close')">
      <div class="wg-modal" @click.stop>
        <header class="wg-head">
          <div class="wg-title">
            <div class="wg-h-ico">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
            </div>
            配置工作台
            <span class="wg-sub">· 选择需要显示的组件，开关实时生效</span>
          </div>
          <button class="wg-reset" @click="reset">恢复默认</button>
          <button class="wg-close" @click="$emit('close')">×</button>
        </header>

        <div class="wg-body">
          <div class="wg-grid">
            <article
              v-for="w in WIDGETS"
              :key="w.code"
              class="wg-card"
              :class="{ 'is-on': isVisible(w.code) }"
              @click="toggle(w.code)"
            >
              <div class="wg-card-head">
                <div class="wg-card-ico" :style="{ background: w.sourceApp.color }">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path :d="w.icon"/></svg>
                </div>
                <div class="wg-card-title">
                  <div class="wg-card-name">{{ w.name }}</div>
                  <div class="wg-card-src">来自 {{ w.sourceApp.name }}</div>
                </div>
                <span class="wg-switch" :class="{ on: isVisible(w.code) }">
                  <span class="wg-switch-thumb"></span>
                </span>
              </div>

              <!-- mini preview thumbnail -->
              <div class="wg-preview" :class="`pv-${w.preview}`">
                <template v-if="w.preview === 'list'">
                  <div class="pv-line" v-for="i in 3" :key="i">
                    <span class="pv-dot" :style="{ background: w.sourceApp.color }"></span>
                    <span class="pv-bar" :style="{ width: 30 + i * 18 + '%' }"></span>
                  </div>
                </template>
                <template v-else-if="w.preview === 'badges'">
                  <div class="pv-row" v-for="i in 2" :key="i">
                    <span class="pv-badge" :style="{ background: w.sourceApp.color }"></span>
                    <span class="pv-bar" :style="{ width: 40 + i * 20 + '%' }"></span>
                  </div>
                </template>
                <template v-else-if="w.preview === 'grid'">
                  <div class="pv-grid">
                    <span class="pv-tile" v-for="i in 4" :key="i" :style="{ background: w.sourceApp.color, opacity: 0.5 + i * 0.1 }"></span>
                  </div>
                </template>
              </div>

              <div class="wg-card-desc">{{ w.description }}</div>
            </article>
          </div>
        </div>

        <footer class="wg-foot">
          <span class="wg-foot-meta">{{ visibleCount }} / {{ WIDGETS.length }} 个组件已启用</span>
          <button class="wg-done" @click="$emit('close')">完成</button>
        </footer>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { WIDGETS } from "./types";
import { useWidgetPrefs } from "./prefs";

defineProps<{ open: boolean }>();
defineEmits<{ (e: "close"): void }>();

const { isVisible, toggle, reset } = useWidgetPrefs();
const visibleCount = computed(() => WIDGETS.filter((w) => isVisible(w.code)).length);
</script>

<style scoped>
.wg-overlay {
  position: fixed; inset: 0;
  background: rgba(13,27,46,.45);
  backdrop-filter: blur(4px);
  display: grid; place-items: center;
  z-index: 1000;
  animation: fadein 0.2s;
}
@keyframes fadein { from { opacity: 0; } to { opacity: 1; } }

.wg-modal {
  background: var(--surface);
  border-radius: 16px;
  box-shadow: var(--sh-4);
  width: min(880px, 92vw);
  max-height: 88vh;
  display: flex; flex-direction: column;
  overflow: hidden;
}
.wg-head {
  display: flex; align-items: center; gap: 14px;
  padding: 16px 22px;
  border-bottom: 1px solid var(--border);
}
.wg-title {
  display: flex; align-items: center; gap: 10px;
  font-size: 15px; font-weight: 600;
  color: var(--text);
  flex: 1;
}
.wg-h-ico {
  width: 26px; height: 26px;
  border-radius: 7px;
  display: grid; place-items: center;
  background: linear-gradient(135deg, var(--primary), var(--purple, #7c3aed));
  color: #fff;
}
.wg-sub {
  font-weight: 400;
  font-size: 12px;
  color: var(--text-3);
}
.wg-reset {
  border: 1px solid var(--border);
  background: var(--surface);
  border-radius: 8px;
  padding: 6px 10px;
  font-size: 12px;
  color: var(--text-2);
  cursor: pointer;
}
.wg-reset:hover { border-color: var(--border-strong); color: var(--text); }
.wg-close {
  width: 28px; height: 28px;
  border: 0; background: var(--surface-2);
  border-radius: 8px;
  font-size: 18px; line-height: 1;
  color: var(--text-3);
  cursor: pointer;
}
.wg-close:hover { background: var(--border); color: var(--text); }

.wg-body {
  padding: 22px;
  overflow-y: auto;
  background: var(--bg-deep, var(--surface-2));
}
.wg-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 14px;
}

.wg-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  padding: 14px;
  cursor: pointer;
  transition: all 0.18s;
  display: flex; flex-direction: column; gap: 10px;
  position: relative;
}
.wg-card:hover {
  border-color: var(--border-strong);
  box-shadow: var(--sh-2);
  transform: translateY(-1px);
}
.wg-card.is-on {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-soft, rgba(30,95,217,.10));
}

.wg-card-head { display: flex; align-items: center; gap: 10px; }
.wg-card-ico {
  width: 32px; height: 32px;
  border-radius: 9px;
  display: grid; place-items: center;
  color: #fff;
  box-shadow: 0 3px 6px rgba(13,27,46,.14);
  flex-shrink: 0;
}
.wg-card-title { flex: 1; min-width: 0; }
.wg-card-name {
  font-size: 13.5px; font-weight: 600;
  color: var(--text);
  line-height: 1.2;
}
.wg-card-src {
  font-size: 11px; color: var(--text-3);
  margin-top: 2px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.wg-switch {
  width: 32px; height: 18px;
  border-radius: 999px;
  background: var(--border);
  position: relative;
  transition: background 0.2s;
  flex-shrink: 0;
}
.wg-switch.on { background: var(--success); }
.wg-switch-thumb {
  position: absolute; top: 2px; left: 2px;
  width: 14px; height: 14px;
  border-radius: 50%;
  background: #fff;
  transition: transform 0.2s;
  box-shadow: 0 1px 3px rgba(13,27,46,.20);
}
.wg-switch.on .wg-switch-thumb { transform: translateX(14px); }

/* Mini preview thumbnails */
.wg-preview {
  background: var(--bg-deep, #f6f8fc);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 10px 12px;
  display: flex; flex-direction: column;
  gap: 5px;
  min-height: 62px;
}
.pv-line, .pv-row { display: flex; align-items: center; gap: 6px; height: 12px; }
.pv-dot {
  width: 5px; height: 5px;
  border-radius: 50%;
  opacity: 0.7;
  flex-shrink: 0;
}
.pv-bar {
  height: 4px;
  background: var(--border-strong);
  border-radius: 2px;
  opacity: 0.55;
}
.pv-badge {
  width: 16px; height: 8px;
  border-radius: 3px;
  opacity: 0.6;
  flex-shrink: 0;
}
.pv-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 5px;
  height: 42px;
}
.pv-tile {
  border-radius: 5px;
  opacity: 0.6;
}

.wg-card-desc {
  font-size: 11.5px;
  color: var(--text-3);
  line-height: 1.5;
}

.wg-foot {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 22px;
  border-top: 1px solid var(--border);
  background: var(--surface);
}
.wg-foot-meta {
  font-size: 12.5px;
  color: var(--text-3);
}
.wg-done {
  background: var(--primary);
  color: #fff;
  border: 0;
  border-radius: 8px;
  padding: 8px 18px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.wg-done:hover { background: var(--primary-deep, var(--primary)); filter: brightness(1.05); }
</style>
