import { defineStore } from "pinia";
import { client } from "@/api/client";
import type { MenuNode } from "@/shell/appcenter/appstore";
import { hasPerm } from "./perm";

interface User { id: string; email?: string; username?: string; phone?: string; password_must_change?: boolean }
interface Tenant { id: string; slug: string; name: string }
interface Token { access_token: string; refresh_token: string; access_expires_at: string }

// MenuTreeNode = a server MenuNode plus its nested children (the /me/menus +
// /admin/menus + /platform/menus APIs return the tree already assembled).
export interface MenuTreeNode extends MenuNode {
  children: MenuTreeNode[];
}

type Console = "tenant" | "platform";

export const useAuthStore = defineStore("auth", {
  state: () => ({
    accessToken: localStorage.getItem("iop.access_token") ?? "",
    refreshToken: localStorage.getItem("iop.refresh_token") ?? "",
    user: null as User | null,
    tenant: null as Tenant | null,
    tenants: [] as Tenant[],
    isPlatformAdmin: false,
    hasPlatformAccess: false,
    isTenantAdmin: false,
    restored: false, // becomes true once restore() has run this page-load
    // RBAC: visible menu trees per console (driven by /me/menus) + flat perm set.
    menusTenant: [] as MenuTreeNode[],
    menusPlatform: [] as MenuTreeNode[],
    permsTenant: [] as string[],
    permsPlatform: [] as string[],
    // perms is the MERGED set across consoles — what v-perm checks against, so a
    // button gates correctly regardless of which console it renders in.
    perms: [] as string[],
  }),
  getters: {
    loggedIn: (s) => !!s.accessToken,
    inTenant: (s) => !!s.tenant,
    // hasPerm(key) — wildcard-aware ("*:*" / "res:*" / exact) check against the
    // merged perm set. Use in components/guards; v-perm uses the same helper.
    hasPerm: (s) => (key: string) => hasPerm(s.perms, key),
  },
  actions: {
    async login(username: string, password: string) {
      const res = await client.post("/auth/login", { username, password });
      const tok = (res.data.data?.token ?? res.data.token) as Token;
      this.user = res.data.data?.user ?? res.data.user;
      this.setToken(tok.access_token, tok.refresh_token);
      await this.loadMyTenants();
      await this.loadAdminFlags(); // works without a tenant (global platform_admin)
      await this.loadMenus();
      await this.loadPerms();
      this.restored = true;
    },
    // restore rehydrates user + tenants from the surviving token after a full page
    // reload (only tokens are persisted to localStorage, not user/tenant objects).
    // Idempotent: safe to call from a route guard on every navigation.
    async restore() {
      if (this.restored) return;
      this.restored = true;
      if (!this.accessToken || this.user) return;
      try {
        const res = await client.get("/me");
        this.user = res.data.data?.user ?? null;
        const preferred = res.data.data?.tenant_id || localStorage.getItem("iop.tenant_id") || "";
        await this.loadMyTenants(preferred);
        await this.loadAdminFlags();
        await this.loadMenus();
        await this.loadPerms();
      } catch {
        // Token invalid / expired — drop it so the guard sends the user to /login.
        this.clearSession();
      }
    },
    async logout() {
      try { await client.post("/auth/logout"); } catch {}
      this.clearSession();
    },
    clearSession() {
      this.accessToken = "";
      this.refreshToken = "";
      this.user = null;
      this.tenant = null;
      this.tenants = [];
      this.isPlatformAdmin = false;
      this.hasPlatformAccess = false;
      this.isTenantAdmin = false;
      this.menusTenant = [];
      this.menusPlatform = [];
      this.permsTenant = [];
      this.permsPlatform = [];
      this.perms = [];
      localStorage.removeItem("iop.access_token");
      localStorage.removeItem("iop.refresh_token");
      localStorage.removeItem("iop.tenant_id");
    },
    // loadAdminFlags refreshes admin flags. is_platform_admin is GLOBAL (works with
    // no tenant); is_tenant_admin reflects the current tenant. Best-effort.
    async loadAdminFlags() {
      try {
        const res = await client.get("/me/admin");
        const d = res.data?.data ?? {};
        this.isPlatformAdmin = !!d.is_platform_admin;
        this.hasPlatformAccess = !!(d.has_platform_access ?? d.is_platform_admin);
        this.isTenantAdmin = !!d.is_tenant_admin;
      } catch {
        this.isPlatformAdmin = false;
        this.hasPlatformAccess = false;
        this.isTenantAdmin = false;
      }
    },
    // loadMenus loads the visible menu tree for BOTH consoles (GET /me/menus).
    // The tenant tree reflects the current tenant context (app enablement +
    // member perms); the platform tree reflects platform-role policies. Best-effort:
    // a non-platform user simply gets an empty platform tree.
    async loadMenus() {
      this.menusTenant = await this.fetchMenus("tenant");
      this.menusPlatform = await this.fetchMenus("platform");
    },
    async fetchMenus(console: Console): Promise<MenuTreeNode[]> {
      try {
        const r = await client.get("/me/menus", { params: { console } });
        return (r.data?.data?.menus ?? []) as MenuTreeNode[];
      } catch {
        return [];
      }
    },
    // loadPerms loads the flat perm set for BOTH consoles (GET /me/perms) and
    // stores a merged set in `perms` (what v-perm / hasPerm check). Per-console
    // sets are kept too in case a caller needs console-scoped gating.
    async loadPerms() {
      this.permsTenant = await this.fetchPerms("tenant");
      this.permsPlatform = await this.fetchPerms("platform");
      this.perms = Array.from(new Set([...this.permsTenant, ...this.permsPlatform]));
    },
    async fetchPerms(console: Console): Promise<string[]> {
      try {
        const r = await client.get("/me/perms", { params: { console } });
        return (r.data?.data?.perms ?? []) as string[];
      } catch {
        return [];
      }
    },
    setToken(access: string, refresh: string) {
      this.accessToken = access;
      this.refreshToken = refresh;
      localStorage.setItem("iop.access_token", access);
      localStorage.setItem("iop.refresh_token", refresh);
    },
    async loadMyTenants(preferredTenantId?: string) {
      const res = await client.get("/me/tenants");
      this.tenants = res.data.data?.tenants ?? [];
      const wanted = preferredTenantId || localStorage.getItem("iop.tenant_id") || "";
      if (wanted) {
        const t = this.tenants.find((x) => x.id === wanted);
        if (t) await this.switchTenant(t.id);
      }
    },
    async switchTenant(tenantId: string) {
      const res = await client.post("/auth/switch-tenant", { tenant_id: tenantId });
      const tok = (res.data.data?.token ?? res.data.token) as Token;
      this.setToken(tok.access_token, tok.refresh_token);
      this.tenant = this.tenants.find((t) => t.id === tenantId) ?? null;
      localStorage.setItem("iop.tenant_id", tenantId);
      await this.loadAdminFlags();
      // Tenant context changed — refresh tenant-scoped menus/perms.
      await this.loadMenus();
      await this.loadPerms();
    },
  },
});
