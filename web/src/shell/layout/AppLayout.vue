<template>
  <div class="app-layout">
    <NavBar />
    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import NavBar from "./NavBar.vue";
</script>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}
.app-main {
  flex: 1;
  padding: var(--space-6);
}
</style>
