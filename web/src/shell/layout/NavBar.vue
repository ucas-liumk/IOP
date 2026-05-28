<template>
  <header class="navbar">
    <div class="brand">IOP</div>
    <nav class="links">
      <router-link to="/">工作台</router-link>
    </nav>
    <div class="env-badge">M1 基座</div>
  </header>
</template>

<style scoped>
.navbar {
  display: flex;
  align-items: center;
  gap: var(--space-6);
  padding: var(--space-3) var(--space-6);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  box-shadow: var(--shadow-sm);
}
.brand {
  font-weight: 600;
  font-size: 18px;
  color: var(--color-primary);
}
.links {
  display: flex;
  gap: var(--space-4);
  flex: 1;
}
.env-badge {
  padding: 2px 8px;
  background: var(--color-bg);
  border-radius: 4px;
  color: var(--color-text-muted);
  font-size: 12px;
}
</style>
