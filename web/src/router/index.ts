import { createRouter, createWebHistory } from 'vue-router';
import { authGuard } from '@/router/guard';
import { routes } from '@/router/routes';

/**
 * router is the main router of the application. The root component sets the document title,
 * since an afterEach hook cannot react to the locale resolving after the first navigation.
 */
export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});

router.beforeEach(authGuard);
