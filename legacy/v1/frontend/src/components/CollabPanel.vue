<script setup lang="ts">
import { ref } from 'vue'
import { ElInput, ElButton, ElIcon, ElMessage } from 'element-plus'
import { Promotion, ChatLineRound, Paperclip } from '@element-plus/icons-vue'
import AvatarBadge from '@/components/AvatarBadge.vue'
import { messagesApi } from '@/api'
import type { Message } from '@/types'

const props = defineProps<{ problemId: string; messages: Message[]; participants: string[] }>()
const emit = defineEmits<{ (e: 'refresh'): void }>()

const text = ref('')
const sending = ref(false)

async function send() {
  if (!text.value.trim()) return
  sending.value = true
  try {
    await messagesApi.post(props.problemId, text.value)
    text.value = ''
    ElMessage.success('已发送')
    emit('refresh')
  } finally { sending.value = false }
}
</script>

<template>
  <div class="form-shell">
    <div class="form-shell-header">
      <div>
        <h2 class="form-shell-title">协同留言</h2>
        <div class="form-shell-sub">@部门或个人，留言会推送到对方工作台。</div>
      </div>
      <div class="row gap-3">
        <span v-for="d in participants" :key="d" class="chip">
          <AvatarBadge :name="d" :size="16" />
          {{ d }}
        </span>
      </div>
    </div>
    <div class="form-shell-body">
      <div v-if="messages.length === 0" style="padding: 32px; text-align: center; color: var(--text-3);">
        <el-icon :size="28"><ChatLineRound /></el-icon>
        <div style="margin-top: 8px;">暂无留言。@相关方开始协同。</div>
      </div>
      <div v-else class="col gap-3">
        <div v-for="m in messages" :key="m.id" style="display: flex; gap: 10px;">
          <AvatarBadge :name="m.actorName" :size="32" />
          <div style="flex: 1; background: var(--surface-3); padding: 10px 14px; border-radius: 12px;">
            <div class="row items-center gap-2">
              <b>{{ m.actorName }}</b>
              <span class="text-xs text-muted">{{ m.occurredAt }}</span>
            </div>
            <div style="font-size: 13.5px; line-height: 1.55; margin-top: 4px;">{{ m.content }}</div>
          </div>
        </div>
      </div>
    </div>
    <div class="form-shell-footer" style="flex-direction: column; align-items: stretch; gap: 8px;">
      <el-input v-model="text" type="textarea" :rows="2" placeholder="@部门 / @人员 写下留言…" />
      <div class="row items-center gap-2">
        <el-button text :icon="Paperclip">附件</el-button>
        <div class="flex-1" />
        <el-button type="primary" :icon="Promotion" :loading="sending" @click="send">发送</el-button>
      </div>
    </div>
  </div>
</template>
