// Each business module exposes a manifest describing itself to the platform.
// The router auto-discovers these via import.meta.glob.
export const manifest = {
  code: "okr",
  name: "OKR 工作安排",
  // routePrefix is the URL space owned by this module.
  // Any route in routes.ts will be mounted under this prefix.
  routePrefix: "/okr",
  // homeRoute is where the left rail navigates to when this app is clicked.
  homeRoute: "/okr/plans",
};
