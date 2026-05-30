// LocalStorage-backed widget visibility prefs.
// Reactive — call useWidgetPrefs() to get a ref + toggle helper.

import { ref, watch } from "vue";
import { WIDGETS, type WidgetCode } from "./types";

const STORAGE_KEY = "iop.workspace.widgets.v1";

// Default: every widget on, in declaration order.
const DEFAULT_VISIBLE: WidgetCode[] = WIDGETS.map((w) => w.code);

interface Prefs {
  visible: WidgetCode[]; // order matters — used as render order
}

function load(): Prefs {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return { visible: [...DEFAULT_VISIBLE] };
    const parsed = JSON.parse(raw) as Partial<Prefs>;
    const visible = Array.isArray(parsed.visible)
      ? parsed.visible.filter((c): c is WidgetCode => DEFAULT_VISIBLE.includes(c as WidgetCode))
      : [...DEFAULT_VISIBLE];
    return { visible };
  } catch {
    return { visible: [...DEFAULT_VISIBLE] };
  }
}

// Singleton — share one reactive prefs across all consumers.
const prefs = ref<Prefs>(load());

watch(
  prefs,
  (v) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(v));
    } catch {}
  },
  { deep: true },
);

export function useWidgetPrefs() {
  function isVisible(code: WidgetCode): boolean {
    return prefs.value.visible.includes(code);
  }
  function toggle(code: WidgetCode) {
    if (isVisible(code)) {
      prefs.value.visible = prefs.value.visible.filter((c) => c !== code);
    } else {
      const next = [...prefs.value.visible, code];
      next.sort((a, b) => DEFAULT_VISIBLE.indexOf(a) - DEFAULT_VISIBLE.indexOf(b));
      prefs.value.visible = next;
    }
  }
  function reset() {
    prefs.value.visible = [...DEFAULT_VISIBLE];
  }
  return { prefs, isVisible, toggle, reset };
}
