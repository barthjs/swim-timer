import { createRouter, createWebHistory } from 'vue-router';
import { i18n } from '@/i18n';

import OverlayLayout from '@/layouts/OverlayLayout.vue';
import WebLayout from '@/layouts/WebLayout.vue';

const routes = [
  {
    path: '/',
    component: WebLayout,
    children: [
      {
        path: '',
        name: 'home',
        component: () => import('@/pages/HomePage.vue'),
      },
      {
        path: '/config',
        name: 'config',
        component: () => import('@/pages/ConfigPage.vue'),
        meta: { requiresAuth: true, title: 'config.title' },
      },
      {
        path: '/login',
        name: 'login',
        component: () => import('@/pages/LoginPage.vue'),
        meta: { isGuest: true, title: 'login.title' },
      },
      {
        path: '/race-data',
        name: 'race-data',
        component: () => import('@/pages/RaceDataPage.vue'),
        meta: { requiresAuth: true, title: 'raceData.title' },
      },
    ],
  },
  {
    path: '/editor',
    name: 'editor',
    component: () => import('@/pages/EditorPage.vue'),
    meta: { requiresAuth: true, title: 'editor.title' },
  },
  {
    path: '/race-overview',
    name: 'race-overview',
    component: () => import('@/pages/RaceOverview.vue'),
    meta: { title: 'raceOverview.title' },
  },
  {
    path: '/overlays',
    component: OverlayLayout,
    children: [
      {
        path: 'ar/:preset',
        name: 'overlay-ar',
        component: () => import('@/pages/overlays/AROverlay.vue'),
        meta: { title: 'overlays.ar' },
      },
      {
        path: 'info',
        name: 'overlay-info',
        component: () => import('@/pages/overlays/InfoOverlay.vue'),
        meta: { title: 'overlays.info' },
      },
      {
        path: 'lane/:number',
        name: 'overlay-lane',
        component: () => import('@/pages/overlays/LaneOverlay.vue'),
        meta: { title: 'overlays.lane' },
      },
      {
        path: 'splits',
        name: 'overlay-splits',
        component: () => import('@/pages/overlays/SplitsOverlay.vue'),
        meta: { title: 'overlays.splits' },
      },
      {
        path: 'startlist',
        name: 'overlay-startlist',
        component: () => import('@/pages/overlays/StartListOverlay.vue'),
        meta: { title: 'overlays.startlist' },
      },
      {
        path: 'timer',
        name: 'overlay-timer',
        component: () => import('@/pages/overlays/TimerOverlay.vue'),
        meta: { title: 'overlays.timer' },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: { name: 'home' },
  },
];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});

router.beforeEach(async (to) => {
  if (to.meta.requiresAuth && sessionStorage.getItem('secretKey') === null) {
    return {
      name: 'login',
      query: { redirect: to.fullPath },
    };
  }

  if (to.meta.isGuest && sessionStorage.getItem('secretKey') !== null) {
    return {
      name: 'home',
    };
  }

  return true;
});

router.afterEach((to) => {
  const { t } = i18n.global;
  const baseTitle = 'Swim-Timer';

  const name = to.name as string;

  switch (name) {
    case 'home':
      document.title = baseTitle;
      break;

    case 'overlay-lane':
      document.title = `${t(to.meta.title as string, { number: to.params.number })} - ${baseTitle}`;
      break;

    default:
      document.title = to.meta.title ? `${t(to.meta.title as string)} - ${baseTitle}` : baseTitle;
      break;
  }
});

export default router;
