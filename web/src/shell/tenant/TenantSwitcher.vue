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
select {
  appearance: none;
  -webkit-appearance: none;
  padding: 4px 24px 4px 0;
  border: 0;
  background: transparent;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-2);
  cursor: pointer;
  background-image: linear-gradient(45deg, transparent 50%, var(--text-3) 50%),
                    linear-gradient(135deg, var(--text-3) 50%, transparent 50%);
  background-position: calc(100% - 12px) 50%, calc(100% - 7px) 50%;
  background-size: 5px 5px, 5px 5px;
  background-repeat: no-repeat;
}
select:focus { outline: none; color: var(--primary); }
</style>
