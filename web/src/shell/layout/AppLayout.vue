<template>
  <div class="app-layout">
    <TopBar />
    <div class="shell">
      <LeftRail />
      <main class="app-main">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import TopBar from "./TopBar.vue";
import LeftRail from "./LeftRail.vue";
</script>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}
.shell {
  display: grid;
  grid-template-columns: var(--rail-w) 1fr;
  min-height: calc(100vh - 56px);
}
.app-main {
  padding: 22px 28px 40px;
  min-width: 0;
  background: var(--bg);
}
</style>
