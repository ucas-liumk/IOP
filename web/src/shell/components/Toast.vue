<template>
  <Teleport to="body">
    <div class="toast-stack" role="region" aria-live="polite" aria-label="通知">
      <TransitionGroup name="toast">
        <div
          v-for="t in toasts"
          :key="t.id"
          class="toast"
          :class="`is-${t.type}`"
          role="status"
        >
          <span class="t-icon" aria-hidden="true">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
              <template v-if="t.type === 'success'">
                <path d="M20 6 9 17l-5-5" />
              </template>
              <template v-else-if="t.type === 'error'">
                <circle cx="12" cy="12" r="9" />
                <line x1="15" y1="9" x2="9" y2="15" />
                <line x1="9" y1="9" x2="15" y2="15" />
              </template>
              <template v-else-if="t.type === 'warning'">
                <path d="M10.3 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.7 3.86a2 2 0 0 0-3.42 0Z" />
                <line x1="12" y1="9" x2="12" y2="13" />
                <circle cx="12" cy="17" r=".6" fill="currentColor" stroke="none" />
              </template>
              <template v-else>
                <circle cx="12" cy="12" r="9" />
                <line x1="12" y1="11" x2="12" y2="16" />
                <circle cx="12" cy="8" r=".6" fill="currentColor" stroke="none" />
              </template>
            </svg>
          </span>
          <span class="t-msg">{{ t.message }}</span>
          <button class="t-close" aria-label="关闭" @click="dismiss(t.id)">×</button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { toasts, dismiss } from "@/shell/notify";
</script>

<style scoped>
.toast-stack {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 9000;
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 360px;
  max-width: calc(100vw - 40px);
  pointer-events: none;
}
.toast {
  pointer-events: auto;
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 12px 12px 14px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-left: 3px solid var(--neutral);
  border-radius: var(--r-md);
  box-shadow: var(--sh-3);
  font-size: 13px;
  line-height: 1.5;
  color: var(--text);
}
.t-icon {
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  border-radius: 7px;
  display: grid;
  place-items: center;
  margin-top: 1px;
}
.t-msg {
  flex: 1;
  min-width: 0;
  white-space: pre-line;
  word-break: break-word;
  padding-top: 1px;
}
.t-close {
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  border: 0;
  background: transparent;
  color: var(--text-4);
  font-size: 18px;
  line-height: 1;
  border-radius: 6px;
  cursor: pointer;
  display: grid;
  place-items: center;
  transition: background .12s, color .12s;
}
.t-close:hover { background: var(--surface-3); color: var(--text-2); }

/* Variants */
.toast.is-success { border-left-color: var(--success); }
.toast.is-success .t-icon { background: var(--success-soft); color: var(--success); }
.toast.is-error { border-left-color: var(--danger); }
.toast.is-error .t-icon { background: var(--danger-soft); color: var(--danger); }
.toast.is-warning { border-left-color: var(--warning); }
.toast.is-warning .t-icon { background: var(--warning-soft); color: var(--warning); }
.toast.is-info { border-left-color: var(--info); }
.toast.is-info .t-icon { background: var(--info-soft); color: var(--info); }

/* Transitions */
.toast-enter-active,
.toast-leave-active {
  transition: transform .28s cubic-bezier(.22, 1, .36, 1), opacity .28s ease;
}
.toast-enter-from {
  transform: translateX(24px);
  opacity: 0;
}
.toast-leave-to {
  transform: translateX(24px);
  opacity: 0;
}
.toast-leave-active {
  position: absolute;
  right: 0;
  width: 100%;
}
.toast-move {
  transition: transform .28s cubic-bezier(.22, 1, .36, 1);
}
</style>
