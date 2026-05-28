<script setup lang="ts">
import { onMounted } from 'vue'
import AppLayout from '@/layouts/AppLayout.vue'
import { useUserStore } from '@/stores/user'
import { useNotificationStore } from '@/stores/notification'

const userStore = useUserStore()
const notif = useNotificationStore()

onMounted(async () => {
  await userStore.loadMe()
  notif.startPolling()
})
</script>

<template>
  <AppLayout />
</template>
