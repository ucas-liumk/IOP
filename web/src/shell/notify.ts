// Global toast/notification system — framework-light singleton store.
// Toast.vue reads `toasts`; any module calls `useNotification()` to push.
import { reactive } from "vue";

export type ToastType = "success" | "error" | "warning" | "info";

export interface Toast {
  id: number;
  type: ToastType;
  message: string;
  /** ms before auto-dismiss; 0 = sticky (manual close only). */
  timeout: number;
}

const DEFAULT_TIMEOUT = 3500;

// Single reactive array shared by the (single) Toast.vue instance.
export const toasts = reactive<Toast[]>([]);

let seq = 0;

export function dismiss(id: number): void {
  const i = toasts.findIndex((t) => t.id === id);
  if (i !== -1) toasts.splice(i, 1);
}

function push(type: ToastType, message: string, timeout = DEFAULT_TIMEOUT): number {
  const id = ++seq;
  toasts.push({ id, type, message, timeout });
  if (timeout > 0) {
    window.setTimeout(() => dismiss(id), timeout);
  }
  return id;
}

export function useNotification() {
  return {
    success: (message: string, timeout?: number) => push("success", message, timeout),
    error: (message: string, timeout?: number) => push("error", message, timeout),
    warning: (message: string, timeout?: number) => push("warning", message, timeout),
    info: (message: string, timeout?: number) => push("info", message, timeout),
    dismiss,
  };
}
