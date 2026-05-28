import { defineStore } from 'pinia'
import { ref } from 'vue'
import { usersApi } from '@/api'
import type { User } from '@/types'

export const useUserStore = defineStore('user', () => {
  const me = ref<{ id: number; name: string; dept: string } | null>(null)
  const all = ref<User[]>([])

  async function loadMe() {
    me.value = await usersApi.me()
  }

  async function loadAll() {
    if (all.value.length === 0) all.value = await usersApi.list()
    return all.value
  }

  function switchUser(id: number) {
    localStorage.setItem('gallant-mock-uid', String(id))
    location.reload()
  }

  return { me, all, loadMe, loadAll, switchUser }
})
