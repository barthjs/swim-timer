import type { RouteLocationNormalized, RouteLocationRaw } from 'vue-router';

import { loginLocation } from '@/lib/auth/redirect';
import { useAuthStore } from '@/stores/auth';

/**
 * authGuard sends guests away from pages that need a login
 * and logged-in admins away from pages meant for guests, such as the login page.
 * Public pages pass without loading the session.
 */
export async function authGuard(to: RouteLocationNormalized): Promise<true | RouteLocationRaw> {
  if (!to.meta.requiresAuth && !to.meta.requiresGuest) return true;

  const authStore = useAuthStore();
  await authStore.ensureLoaded();

  // An unknown status lets the navigation through, so a server that is briefly unreachable does not log anyone out.
  // A page that needs a login then fails with its own request, and a real 401 leads to the login page.
  if (to.meta.requiresAuth && authStore.authenticated === false) {
    return loginLocation(
      {
        name: 'login',
      },
      to.fullPath,
    );
  }

  if (to.meta.requiresGuest && authStore.authenticated === true) {
    return {
      name: 'dashboard',
    };
  }

  return true;
}
