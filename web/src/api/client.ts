import axios, { type AxiosInstance, type InternalAxiosRequestConfig } from "axios";

function newId(): string {
  // UUID v4 — sufficient for Idempotency-Key (M2 enforces server-side).
  return crypto.randomUUID();
}

function createClient(): AxiosInstance {
  const baseURL =
    (import.meta.env.VITE_API_BASE_URL as string) ??
    (import.meta.env.MODE === "development" ? "/api" : "/api");

  const instance = axios.create({
    baseURL,
    timeout: 30_000,
    withCredentials: false,
  });

  instance.interceptors.request.use((cfg: InternalAxiosRequestConfig) => {
    const method = (cfg.method ?? "get").toUpperCase();
    if (method !== "GET" && method !== "HEAD") {
      // Idempotency-Key for all mutations. M2 server-side middleware caches replies.
      cfg.headers.set("Idempotency-Key", newId());
    }
    // X-Request-Id for trace propagation. Server echoes via RequestID middleware.
    cfg.headers.set("X-Request-Id", newId());
    // Auth: attach access token if present.  Skip /auth/login + /auth/refresh.
    const skip = cfg.url?.endsWith("/auth/login") || cfg.url?.endsWith("/auth/refresh");
    if (!skip) {
      const tok = localStorage.getItem("iop.access_token");
      if (tok) cfg.headers.set("Authorization", "Bearer " + tok);
    }
    return cfg;
  });

  instance.interceptors.response.use(
    (res) => res,
    (err) => {
      if (err.response?.status === 401 && !err.config.url?.endsWith("/auth/login")) {
        // Drop token; redirect to login.  Real refresh flow goes here in M3+.
        localStorage.removeItem("iop.access_token");
        localStorage.removeItem("iop.refresh_token");
        if (location.pathname !== "/login") location.href = "/login";
      }
      return Promise.reject(err);
    },
  );

  return instance;
}

export const client = createClient();
