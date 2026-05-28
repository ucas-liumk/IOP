import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { notificationsApi } from '@/api'
import type { NotificationDigest } from '@/types'

const POLL_INTERVAL_MS = 20_000

export const useNotificationStore = defineStore('notification', () => {
  const digest = ref<NotificationDigest>({ mentions: [], overdues: [], dueSoon: [] })
  const total = computed(() => digest.value.mentions.length + digest.value.overdues.length + digest.value.dueSoon.length)
  let timer: ReturnType<typeof setInterval> | null = null

  async function refresh() {
    try {
      digest.value = await notificationsApi.unread()
    } catch {
      /* silently ignore - polling will retry */
    }
  }

  function startPolling() {
    refresh()
    if (timer) clearInterval(timer)
    timer = setInterval(refresh, POLL_INTERVAL_MS)
  }

  function stopPolling() {
    if (timer) clearInterval(timer)
    timer = null
  }

  return { digest, total, refresh, startPolling, stopPolling }
})
