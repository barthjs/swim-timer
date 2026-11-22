import type { RouteLocationNormalizedLoaded, RouteRecordRaw } from 'vue-router';
import AdminLayout from '@/layouts/AdminLayout.vue';
import AppLayout from '@/layouts/AppLayout.vue';

declare module 'vue-router' {
  // eslint-disable-next-line @typescript-eslint/consistent-type-definitions -- augmentation merges into the interface of the library
  interface RouteMeta {
    titleKey?: string;
    // Resolves the interpolation values of the title from the route
    titleParams?: (route: RouteLocationNormalizedLoaded) => Record<string, string>;
    requiresAuth?: boolean;
    requiresGuest?: boolean;
    // From lg up, the page fills the height of the panel instead of scrolling,
    // for a page with sections only while the route names a section that needs it.
    fillViewport?: boolean | ((route: RouteLocationNormalizedLoaded) => boolean);
    // Nobody reads the page interactively, so it loads a new version as soon as nothing on it would be lost
    unattended?: boolean;
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
        component: async () => import('@/pages/LoginPage.vue'),
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
        component: async () => import('@/pages/DashboardPage.vue'),
      },
      {
        path: 'meet',
        name: 'meet',
        meta: {
          titleKey: 'pages.meet.title',
          fillViewport: true,
        },
        component: async () => import('@/pages/MeetPage.vue'),
      },
      {
        path: 'meet-data',
        name: 'meetData',
        meta: {
          titleKey: 'pages.meetData.title',
          requiresAuth: true,
        },
        component: async () => import('@/pages/MeetDataPage.vue'),
      },
      {
        path: 'records',
        name: 'records',
        meta: {
          titleKey: 'pages.records.title',
          requiresAuth: true,
          fillViewport: true,
        },
        component: async () => import('@/pages/RecordsPage.vue'),
      },
      {
        path: 'editor',
        name: 'editor',
        meta: {
          titleKey: 'pages.editor.title',
          requiresAuth: true,
          fillViewport: true,
        },
        component: async () => import('@/pages/EditorPage.vue'),
      },
      {
        path: 'config',
        name: 'config',
        meta: {
          titleKey: 'pages.config.title',
          requiresAuth: true,
        },
        component: async () => import('@/pages/ConfigPage.vue'),
      },
    ],
  },
  {
    path: '/scoreboard',
    name: 'scoreboard',
    meta: {
      titleKey: 'pages.scoreboard.title',
      unattended: true,
    },
    component: async () => import('@/pages/ScoreboardPage.vue'),
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: {
      name: 'dashboard',
    },
  },
];
