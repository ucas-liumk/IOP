<template>
  <article class="wcard" :class="{ 'is-config': configMode }">
    <header class="whead">
      <div class="wico" :style="{ background: source.color }">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path :d="icon"/></svg>
      </div>
      <div class="wtitle">
        <div class="wname">{{ title }}</div>
        <div class="wsub">来自 {{ source.name }}</div>
      </div>
      <div class="wright">
        <slot name="actions">
          <a v-if="more" :href="more.to" class="wmore" @click.prevent="more.go">{{ more.label }} →</a>
        </slot>
      </div>
    </header>
    <div class="wbody">
      <slot />
    </div>
  </article>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  icon: string;
  source: { code: string; name: string; color: string };
  more?: { label: string; to: string; go: () => void };
  configMode?: boolean;
}>();
</script>

<style scoped>
.wcard {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: var(--sh-1);
  overflow: hidden;
  transition: outline 0.18s, box-shadow 0.18s;
  outline: 2px dashed transparent;
  outline-offset: 2px;
}
.wcard.is-config {
  outline-color: var(--primary);
}
.whead {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border);
}
.wico {
  width: 28px; height: 28px;
  border-radius: 8px;
  display: grid; place-items: center;
  color: #fff;
  box-shadow: 0 2px 5px rgba(13,27,46,.12);
}
.wtitle { flex: 1; min-width: 0; }
.wname {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  line-height: 1.2;
}
.wsub {
  font-size: 11.5px;
  color: var(--text-3);
  margin-top: 2px;
}
.wright {
  font-size: 12.5px;
}
.wmore {
  color: var(--text-3);
  text-decoration: none;
  font-size: 12.5px;
}
.wmore:hover { color: var(--primary); }
.wbody {
  padding: 12px 18px 16px;
}
</style>
