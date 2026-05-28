import type { NavigationGuard } from "vue-router";
import { useAuthStore } from "./auth.store";

export const requireAuth: NavigationGuard = (to, _from, next) => {
  const auth = useAuthStore();
  if (!auth.loggedIn) {
    return next({ name: "login", query: { redirect: to.fullPath } });
  }
  next();
};
