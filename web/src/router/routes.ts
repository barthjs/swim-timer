import type { RouteLocationNormalizedLoaded, RouteRecordRaw } from 'vue-router';
import AdminLayout from '@/layouts/AdminLayout.vue';
import AppLayout from '@/layouts/AppLayout.vue';

declare module 'vue-router' {
  interface RouteMeta {
    titleKey?: string;
    // Resolves the interpolation values of the title from the route.
    titleParams?: (route: RouteLocationNormalizedLoaded) => Record<string, string>;
    requiresAuth?: boolean;
    requiresGuest?: boolean;
    // The page uses the whole width of the main area instead of the container width.
    fullWidth?: boolean;
    // From lg up, the page fills the height between header and footer instead of scrolling,
    // for a page with tabs only while the route names a tab that needs it.
    fillViewport?: boolean | ((route: RouteLocationNormalizedLoaded) => boolean);
  }
}

export const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    component: AppLayout,
    children: [
      {
        path: '',
        name: 'login',
        meta: {
          titleKey: 'pages.login.title',
          requiresGuest: true,
        },
        component: () => import('@/pages/LoginPage.vue'),
      },
    ],
  },
  {
    path: '/',
    component: AdminLayout,
    children: [
      {
        path: '',
        name: 'dashboard',
        meta: {
          titleKey: 'pages.dashboard.title',
        },
        component: () => import('@/pages/DashboardPage.vue'),
      },
      {
        path: 'meet',
        name: 'meet',
        meta: {
          titleKey: 'pages.meet.title',
          fillViewport: true,
        },
        component: () => import('@/pages/MeetPage.vue'),
      },
      {
        path: 'meet-data',
        name: 'meetData',
        meta: {
          titleKey: 'pages.meetData.title',
          requiresAuth: true,
        },
        component: () => import('@/pages/MeetDataPage.vue'),
      },
      {
        path: 'editor',
        name: 'editor',
        meta: {
          titleKey: 'pages.editor.title',
          requiresAuth: true,
          fullWidth: true,
          fillViewport: true,
        },
        component: () => import('@/pages/EditorPage.vue'),
      },
      {
        path: 'config',
        name: 'config',
        meta: {
          titleKey: 'pages.config.title',
          requiresAuth: true,
          // The table of the webhooks reaches down to the footer like the other tables
          fillViewport: (route) => route.query.tab === 'webhook',
        },
        component: () => import('@/pages/ConfigPage.vue'),
      },
    ],
  },
  {
    path: '/scoreboard',
    name: 'scoreboard',
    meta: {
      titleKey: 'pages.scoreboard.title',
    },
    component: () => import('@/pages/ScoreboardPage.vue'),
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: {
      name: 'dashboard',
    },
  },
];
