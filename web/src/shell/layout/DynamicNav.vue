<template>
  <template v-for="node in menus" :key="node.key">
    <!-- Directory node: a group header + recursive children. -->
    <div v-if="node.type === 'dir'" class="nav-group">
      <div class="group-title">{{ node.title }}</div>
      <DynamicNav :menus="node.children" />
    </div>

    <!-- Menu node with a path: a navigable link. -->
    <router-link
      v-else-if="node.path"
      :to="node.path"
      class="nav-link"
      :class="{ active: isActive(node.path) }"
    >
      <svg
        v-if="node.icon"
        width="14" height="14" viewBox="0 0 24 24"
        fill="none" stroke="currentColor" stroke-width="2"
        stroke-linecap="round" stroke-linejoin="round"
      >
        <path :d="node.icon" />
      </svg>
      <span class="nav-label">{{ node.title }}</span>
    </router-link>

    <!-- Menu node without a path but with children (nested grouping): recurse. -->
    <DynamicNav v-else-if="node.children?.length" :menus="node.children" />
  </template>
</template>

<script setup lang="ts">
import { useRoute } from "vue-router";
import type { MenuTreeNode } from "@/shell/auth/auth.store";

defineOptions({ name: "DynamicNav" });
defineProps<{ menus: MenuTreeNode[] }>();

const route = useRoute();

// A link is active when the current path equals it, or is nested under it
// (path + "/"). The "/" guard stops a parent like "/admin" from matching every
// child route; exact-match still highlights the index page.
function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(path + "/");
}
</script>

<style scoped>
.nav-group { display: flex; flex-direction: column; gap: 2px; margin-top: 6px; }
.group-title {
  font-size: 10.5px; font-weight: 600;
  color: var(--text-4); text-transform: uppercase;
  letter-spacing: .8px;
  padding: 4px 8px;
}
.nav-link {
  display: flex; align-items: center; gap: 9px;
  padding: 7px 10px;
  font-size: 13px;
  color: var(--text-2);
  border-radius: 7px;
  text-decoration: none;
  cursor: pointer;
  transition: background .12s, color .12s;
}
.nav-link:hover { background: var(--bg); color: var(--text); text-decoration: none; }
.nav-link.active { background: var(--primary-soft); color: var(--primary); font-weight: 600; }
.nav-link.active svg { color: var(--primary); }
.nav-label { flex: 1; min-width: 0; }
</style>
