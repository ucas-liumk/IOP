import type { NavigationGuard } from "vue-router";
import { useAuthStore } from "./auth.store";

// requireAuth gates protected routes. On a full page reload only the token
// survives in localStorage, so we rehydrate user/tenant from /me before deciding.
export const requireAuth: NavigationGuard = async (to, _from, next) => {
  const auth = useAuthStore();
  if (auth.loggedIn && !auth.restored) {
    await auth.restore();
  }
  if (!auth.loggedIn) {
    return next({ name: "login", query: { redirect: to.fullPath } });
  }
  // Force initial-password change: until done, only the security page is reachable.
  if (auth.user?.password_must_change && !to.path.startsWith("/me/security")) {
    return next({ path: "/me/security", query: { forced: "1" } });
  }
  next();
};

// requirePlatformAdmin is a per-route beforeEnter for platform-only screens
// (用户管理 / 组织机构). Defense-in-depth + UX — the backend PlatformAdminRequired
// middleware remains the authoritative gate; this just avoids rendering a shell
// that would only fire failing requests. Placed on the child route so it re-runs
// on SPA navigation, not just on first entry into the /admin subtree.
export const requirePlatformAdmin: NavigationGuard = async (_to, _from, next) => {
  const auth = useAuthStore();
  if (auth.loggedIn && !auth.restored) {
    await auth.restore();
  }
  if (!auth.isPlatformAdmin) {
    // Not a platform admin → send to their normal landing (workspace / tenant console).
    return next({ path: "/" });
  }
  next();
};

// homeForRole returns where a freshly-authenticated user should land.
export function homeForRole(): string {
  const auth = useAuthStore();
  if (auth.user?.password_must_change) return "/me/security";
  if (auth.isPlatformAdmin) return "/platform";
  return "/";
}

// requireTenantAdmin gates the tenant console (/admin/*). Non-tenant-admins
// (incl. platform admins, who govern at the platform layer) are sent to their
// own home so they never see a tenant console that would only 403.
export const requireTenantAdmin: NavigationGuard = async (_to, _from, next) => {
  const auth = useAuthStore();
  if (auth.loggedIn && !auth.restored) {
    await auth.restore();
  }
  if (!auth.isTenantAdmin) {
    return next({ path: homeForRole() });
  }
  next();
};
