import { createRouter, createWebHistory } from "vue-router";
import { getSession } from "../auth";

import AppLayout from "../layouts/AppLayout.vue";
import LoginView from "../views/LoginView.vue";
import DashboardView from "../views/DashboardView.vue";
import ClientsView from "../views/ClientsView.vue";
import MasterView from "../views/MasterView.vue";
import CamerasView from "../views/CamerasView.vue";
import ViolationsView from "../views/ViolationsView.vue";

const routes = [
  {
    path: "/login",
    name: "login",
    component: LoginView,
    meta: { guestOnly: true },
  },
  {
    path: "/",
    component: AppLayout,
    meta: { requiresAuth: true },
    children: [
      {
        path: "",
        redirect: "/dashboard",
      },
      {
        path: "dashboard",
        name: "dashboard",
        component: DashboardView,
      },
      {
        path: "clients",
        name: "clients",
        component: ClientsView,
      },
      {
        path: "master-pelanggaran",
        name: "master-pelanggaran",
        component: MasterView,
      },
      {
        path: "cameras",
        name: "cameras",
        component: CamerasView,
      },
      {
        path: "violations",
        name: "violations",
        component: ViolationsView,
      },
    ],
  },
  {
    path: "/:pathMatch(.*)*",
    redirect: "/dashboard",
  },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});

let cachedSession: any = null;
let lastCheck = 0;

export async function checkAuth() {
  const now = Date.now();
  if (cachedSession && now - lastCheck < 30000) {
    return cachedSession;
  }
  try {
    const res: any = await getSession();
    const data = res?.data || res;
    if (data?.session && data?.user) {
      cachedSession = data;
      lastCheck = now;
      return cachedSession;
    }
  } catch {
    // ignore
  }
  cachedSession = null;
  return null;
}

export function clearCachedSession() {
  cachedSession = null;
  lastCheck = 0;
}

router.beforeEach(async (to, _from, next) => {
  const session = await checkAuth();
  if (to.meta.requiresAuth && !session) {
    return next({ path: "/login", query: { redirect: to.fullPath } });
  }
  if (to.meta.guestOnly && session) {
    return next({ path: "/dashboard" });
  }
  next();
});
