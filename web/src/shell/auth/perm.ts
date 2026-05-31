// RBAC button-level gating helpers.
//
// Perm keys are "resource:action" strings (e.g. "member:read"). The backend
// already expands wildcards into the user's flat perm set, so a perm set may
// contain "*:*" (full access), "member:*" (resource-wide), or exact keys.
//
// hasPerm() does the inverse client-side match: given the user's granted perm
// set and a required key, is the key satisfied (exactly or via a wildcard)?
import type { DirectiveBinding, ObjectDirective } from "vue";
import { useAuthStore } from "./auth.store";

// hasPerm reports whether `key` ("resource:action") is satisfied by `perms`.
// A key is satisfied when any granted perm matches it, where a granted perm may
// use "*" in either segment: "*:*", "res:*", "*:action", or an exact "res:act".
export function hasPerm(perms: string[], key: string): boolean {
  if (!key) return true; // no requirement => always allowed
  if (!perms || perms.length === 0) return false;
  const [needRes, needAct] = splitKey(key);
  for (const granted of perms) {
    const [gRes, gAct] = splitKey(granted);
    const resOk = gRes === "*" || gRes === needRes;
    const actOk = gAct === "*" || gAct === needAct;
    if (resOk && actOk) return true;
  }
  return false;
}

// splitKey splits "resource:action" on the LAST ":" (resources may contain ":",
// e.g. "okr:plan:read" => resource "okr:plan", action "read"). Mirrors the
// backend's splitPerm. A bare key with no ":" is treated as resource + "*".
function splitKey(k: string): [string, string] {
  const i = k.lastIndexOf(":");
  if (i < 0) return [k, "*"];
  return [k.slice(0, i), k.slice(i + 1)];
}

// vPerm hides/removes an element when the current user's perms don't satisfy the
// binding value. Usage: v-perm="'member:write'". A falsy/empty value never hides.
//
// We remove the element from the DOM (comment placeholder) rather than just
// hiding it, so gated actions can't be reached via devtools `display:none`
// toggling. Re-evaluated on update so reactive perm changes (login/switch) apply.
export const vPerm: ObjectDirective<HTMLElement, string> = {
  mounted(el, binding) {
    apply(el, binding);
  },
  updated(el, binding) {
    apply(el, binding);
  },
};

function apply(el: HTMLElement, binding: DirectiveBinding<string>) {
  const key = binding.value;
  const auth = useAuthStore();
  const allowed = hasPerm(auth.perms, key);
  if (allowed) {
    // Restore if a previous evaluation had removed it.
    const ph = (el as any).__permPlaceholder as Comment | undefined;
    if (ph && ph.parentNode) {
      ph.parentNode.replaceChild(el, ph);
      (el as any).__permPlaceholder = undefined;
    }
    el.style.removeProperty("display");
  } else {
    // Remove from DOM, leaving a comment placeholder to restore later.
    if (!(el as any).__permPlaceholder && el.parentNode) {
      const ph = document.createComment("v-perm");
      (el as any).__permPlaceholder = ph;
      el.parentNode.replaceChild(ph, el);
    }
  }
}
