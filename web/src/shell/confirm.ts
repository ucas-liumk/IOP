// Global confirm-dialog system — singleton store + promise-based API.
// ConfirmDialog.vue renders `pending`; callers `await useConfirm().confirm(...)`.
import { ref } from "vue";

export interface ConfirmOptions {
  title?: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  /** Style the confirm button as destructive (red). */
  danger?: boolean;
}

interface ConfirmRequest extends ConfirmOptions {
  id: number;
  resolve: (ok: boolean) => void;
}

// The single pending request (only one dialog at a time).
export const pending = ref<ConfirmRequest | null>(null);

let seq = 0;

export function useConfirm() {
  function confirm(options: ConfirmOptions): Promise<boolean> {
    // If a dialog is already open, resolve it as cancelled before replacing.
    if (pending.value) pending.value.resolve(false);
    return new Promise<boolean>((resolve) => {
      pending.value = { id: ++seq, ...options, resolve };
    });
  }
  return { confirm };
}

export function resolvePending(ok: boolean): void {
  const req = pending.value;
  if (!req) return;
  pending.value = null;
  req.resolve(ok);
}
