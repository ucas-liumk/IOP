import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import router from "./router";
import { vPerm } from "./shell/auth/perm";
import "./styles/global.css";

const app = createApp(App);
app.use(createPinia());
app.use(router);
// Button-level RBAC gate: v-perm="'resource:action'" removes the element when
// the current user's perms don't satisfy the key. Pinia must be installed first
// (the directive reads the auth store).
app.directive("perm", vPerm);
app.mount("#app");
