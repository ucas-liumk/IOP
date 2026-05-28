<template>
  <select v-if="auth.tenants.length > 0" v-model="selected" @change="onChange">
    <option v-for="t in auth.tenants" :key="t.id" :value="t.id">{{ t.name }}</option>
  </select>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useAuthStore } from "@/shell/auth/auth.store";

const auth = useAuthStore();
const selected = ref(auth.tenant?.id ?? "");

watch(() => auth.tenant?.id, (id) => { selected.value = id ?? ""; });

async function onChange() {
  if (selected.value) await auth.switchTenant(selected.value);
}
</script>

<style scoped>
select { padding: var(--space-1) var(--space-2); border: 1px solid var(--color-border); border-radius: var(--radius); background: var(--color-surface); font-size: 13px; }
</style>
