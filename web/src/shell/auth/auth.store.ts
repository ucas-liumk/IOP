import { defineStore } from "pinia";
import { client } from "@/api/client";

interface User { id: string; email?: string; username?: string; phone?: string; password_must_change?: boolean }
interface Tenant { id: string; slug: string; name: string }
interface Token { access_token: string; refresh_token: string; access_expires_at: string }

export const useAuthStore = defineStore("auth", {
  state: () => ({
    accessToken: localStorage.getItem("iop.access_token") ?? "",
    refreshToken: localStorage.getItem("iop.refresh_token") ?? "",
    user: null as User | null,
    tenant: null as Tenant | null,
    tenants: [] as Tenant[],
    isPlatformAdmin: false,
    isTenantAdmin: false,
    restored: false, // becomes true once restore() has run this page-load
  }),
  getters: {
    loggedIn: (s) => !!s.accessToken,
    inTenant: (s) => !!s.tenant,
  },
  actions: {
    async login(username: string, password: string) {
      const res = await client.post("/auth/login", { username, password });
      const tok = (res.data.data?.token ?? res.data.token) as Token;
      this.user = res.data.data?.user ?? res.data.user;
      this.setToken(tok.access_token, tok.refresh_token);
      await this.loadMyTenants();
      await this.loadAdminFlags(); // works without a tenant (global platform_admin)
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
      this.isTenantAdmin = false;
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
        this.isTenantAdmin = !!d.is_tenant_admin;
      } catch {
        this.isPlatformAdmin = false;
        this.isTenantAdmin = false;
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
    },
  },
});
