<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElBadge, ElButton, ElIcon, ElDropdown, ElDropdownMenu, ElDropdownItem } from 'element-plus'
import { Bell, Plus, Search, Grid, List } from '@element-plus/icons-vue'
import AvatarBadge from '@/components/AvatarBadge.vue'
import ProcessingModal from '@/components/ProcessingModal.vue'
import { useUserStore } from '@/stores/user'
import { useNotificationStore } from '@/stores/notification'
import { useStagesStore } from '@/stores/stages'
import { useProcessingStore } from '@/stores/processing'

const route = useRoute()
const router = useRouter()
const user = useUserStore()
const notif = useNotificationStore()
const stages = useStagesStore()
const proc = useProcessingStore()

const navTabs = [
  { key: 'dashboard', label: '看板',     icon: Grid, route: '/dashboard' },
  { key: 'problems',  label: '问题清单', icon: List, route: '/problems' },
]

const activeKey = computed(() => route.name as string)

onMounted(async () => {
  await Promise.all([stages.load(), user.loadAll()])
})

function newProblem() {
  router.push('/problems')
}
</script>

<template>
  <div class="app">
    <nav class="nav">
      <div class="nav-brand">
        <div class="nav-brand-mark">协</div>
        <div>
          <div style="line-height: 1.1;">协同研究</div>
          <div style="font-size: 11px; color: var(--text-3); font-weight: 500;">问题协同解决平台</div>
        </div>
      </div>
      <div class="nav-tabs">
        <div
          v-for="t in navTabs"
          :key="t.key"
          class="nav-tab"
          :class="{ active: activeKey === t.key }"
          @click="router.push(t.route)"
        >
          <el-icon :size="15"><component :is="t.icon" /></el-icon>
          {{ t.label }}
        </div>
      </div>
      <div class="nav-spacer" />
      <div class="nav-tools">
        <el-button type="primary" size="small" @click="newProblem">
          <el-icon><Plus /></el-icon>&nbsp;提报问题
        </el-button>
        <el-button text :icon="Search" circle />
        <el-badge :value="notif.total" :hidden="notif.total === 0" :max="99">
          <el-button text :icon="Bell" circle />
        </el-badge>
        <el-dropdown>
          <div class="nav-user">
            <AvatarBadge :name="user.me?.name || '?'" :size="28" />
            <span style="font-size: 13px; font-weight: 500;">{{ user.me?.name || '加载中...' }}</span>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-for="u in user.all" :key="u.id" @click="user.switchUser(u.id)">
                <AvatarBadge :name="u.name" :size="20" :color="u.avatarColor" />
                <span style="margin-left: 8px;">{{ u.name }}</span>
                <span class="text-muted text-xs" style="margin-left: 6px;">{{ u.dept }}</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </nav>

    <main class="app-main">
      <router-view />
    </main>

    <ProcessingModal v-if="proc.visible" />
  </div>
</template>

<style scoped>
.app-main { flex: 1; padding: 0; }
</style>
