import { createRouter, createWebHistory } from 'vue-router';
import { i18n } from '@/i18n';

import OverlayLayout from '@/layouts/OverlayLayout.vue';

const routes = [
  {
    path: '/overlays',
    component: OverlayLayout,
    children: [
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

router.afterEach((to) => {
  const { t } = i18n.global;
  const baseTitle = 'Swim-Timer';

  const name = to.name as string;

  switch (name) {
    case 'overlay-lane':
      document.title = `${t(to.meta.title as string, { number: to.params.number })} - ${baseTitle}`;
      break;

    default:
      document.title = to.meta.title ? `${t(to.meta.title as string)} - ${baseTitle}` : baseTitle;
      break;
  }
});

export default router;
