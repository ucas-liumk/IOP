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
  padding: var(--sp-7) var(--sp-7) var(--sp-9);
  max-width: 1280px;
  width: 100%;
  margin: 0 auto;
}
</style>
