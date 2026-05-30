<template>
  <Teleport to="body">
    <Transition name="cd-fade">
      <div
        v-if="pending"
        class="cd-overlay"
        @click.self="onCancel"
        @keydown.esc="onCancel"
      >
        <Transition name="cd-pop" appear>
          <div
            class="cd-modal"
            role="alertdialog"
            aria-modal="true"
            :aria-label="pending.title || '确认'"
          >
            <div class="cd-icon" :class="{ danger: pending.danger }" aria-hidden="true">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
                <template v-if="pending.danger">
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
            </div>
            <h2 class="cd-title">{{ pending.title || "确认" }}</h2>
            <p class="cd-message">{{ pending.message }}</p>
            <div class="cd-actions">
              <button class="btn btn-ghost" @click="onCancel">
                {{ pending.cancelText || "取消" }}
              </button>
              <button
                ref="confirmBtn"
                class="btn"
                :class="pending.danger ? 'btn-danger' : 'btn-primary'"
                @click="onConfirm"
              >
                {{ pending.confirmText || "确定" }}
              </button>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from "vue";
import { pending, resolvePending } from "@/shell/confirm";

const confirmBtn = ref<HTMLButtonElement | null>(null);

function onConfirm() {
  resolvePending(true);
}
function onCancel() {
  resolvePending(false);
}

// Autofocus the confirm button when a dialog opens, for keyboard users.
watch(pending, async (req) => {
  if (req) {
    await nextTick();
    confirmBtn.value?.focus();
  }
});
</script>

<style scoped>
.cd-overlay {
  position: fixed;
  inset: 0;
  z-index: 9100;
  background: rgba(13, 27, 46, .42);
  backdrop-filter: blur(3px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.cd-modal {
  width: 100%;
  max-width: 400px;
  background: var(--surface);
  border-radius: var(--r-lg);
  box-shadow: var(--sh-4);
  padding: 26px 26px 22px;
  text-align: center;
}
.cd-icon {
  width: 46px;
  height: 46px;
  margin: 0 auto 14px;
  border-radius: 13px;
  display: grid;
  place-items: center;
  background: var(--info-soft);
  color: var(--info);
}
.cd-icon.danger {
  background: var(--danger-soft);
  color: var(--danger);
}
.cd-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--text);
  letter-spacing: -.01em;
}
.cd-message {
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-2);
  margin-top: 8px;
  white-space: pre-line;
  word-break: break-word;
}
.cd-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
  margin-top: 22px;
}
.cd-actions .btn { min-width: 92px; justify-content: center; cursor: pointer; }
.btn-danger {
  background: var(--danger);
  color: #fff;
  border-color: var(--danger);
}
.btn-danger:hover { background: #c92f2f; border-color: #c92f2f; }

/* Transitions */
.cd-fade-enter-active,
.cd-fade-leave-active { transition: opacity .2s ease; }
.cd-fade-enter-from,
.cd-fade-leave-to { opacity: 0; }

.cd-pop-enter-active { transition: transform .26s cubic-bezier(.22, 1, .36, 1), opacity .26s ease; }
.cd-pop-enter-from { transform: translateY(10px) scale(.96); opacity: 0; }
</style>
