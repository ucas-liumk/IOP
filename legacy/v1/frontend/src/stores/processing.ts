import { defineStore } from 'pinia'
import { ref } from 'vue'
import { problemsApi } from '@/api'
import type { ProblemDetail } from '@/types'

/** 全局办理弹层状态。任何页面调用 open(id) 即弹出全屏办理界面 */
export const useProcessingStore = defineStore('processing', () => {
  const visible = ref(false)
  const loading = ref(false)
  const detail = ref<ProblemDetail | null>(null)
  const activeProblemId = ref<string | null>(null)

  async function open(id: string) {
    activeProblemId.value = id
    visible.value = true
    await reload()
  }

  async function reload() {
    if (!activeProblemId.value) return
    loading.value = true
    try {
      detail.value = await problemsApi.detail(activeProblemId.value)
    } finally {
      loading.value = false
    }
  }

  function close() {
    visible.value = false
    activeProblemId.value = null
    detail.value = null
  }

  return { visible, loading, detail, activeProblemId, open, reload, close }
})
