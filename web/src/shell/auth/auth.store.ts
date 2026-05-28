import { defineStore } from "pinia";
import { client } from "@/api/client";

interface User { id: string; email: string }
interface Tenant { id: string; slug: string; name: string }
interface Token { access_token: string; refresh_token: string; access_expires_at: string }

export const useAuthStore = defineStore("auth", {
  state: () => ({
    accessToken: localStorage.getItem("iop.access_token") ?? "",
    refreshToken: localStorage.getItem("iop.refresh_token") ?? "",
    user: null as User | null,
    tenant: null as Tenant | null,
    tenants: [] as Tenant[],
  }),
  getters: {
    loggedIn: (s) => !!s.accessToken,
    inTenant: (s) => !!s.tenant,
  },
  actions: {
    async login(email: string, password: string) {
      const res = await client.post("/auth/login", { email, password });
      const tok = (res.data.data?.token ?? res.data.token) as Token;
      this.user = res.data.data?.user ?? res.data.user;
      this.setToken(tok.access_token, tok.refresh_token);
      await this.loadMyTenants();
    },
    async logout() {
      try { await client.post("/auth/logout"); } catch {}
      this.accessToken = "";
      this.refreshToken = "";
      this.user = null;
      this.tenant = null;
      this.tenants = [];
      localStorage.removeItem("iop.access_token");
      localStorage.removeItem("iop.refresh_token");
      localStorage.removeItem("iop.tenant_id");
    },
    setToken(access: string, refresh: string) {
      this.accessToken = access;
      this.refreshToken = refresh;
      localStorage.setItem("iop.access_token", access);
      localStorage.setItem("iop.refresh_token", refresh);
    },
    async loadMyTenants() {
      const res = await client.get("/me/tenants");
      this.tenants = res.data.data?.tenants ?? [];
      const cached = localStorage.getItem("iop.tenant_id");
      if (cached) {
        const t = this.tenants.find((x) => x.id === cached);
        if (t) await this.switchTenant(t.id);
      }
    },
    async switchTenant(tenantId: string) {
      const res = await client.post("/auth/switch-tenant", { tenant_id: tenantId });
      const tok = (res.data.data?.token ?? res.data.token) as Token;
      this.setToken(tok.access_token, tok.refresh_token);
      this.tenant = this.tenants.find((t) => t.id === tenantId) ?? null;
      localStorage.setItem("iop.tenant_id", tenantId);
    },
  },
});
