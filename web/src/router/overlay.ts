import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import OverlayLayout from '@/layouts/OverlayLayout.vue';

const routes: RouteRecordRaw[] = [
  {
    path: '/overlays',
    component: OverlayLayout,
    children: [
      {
        path: 'ar/:preset',
        name: 'overlayAr',
        meta: {
          titleKey: 'overlays.ar',
          titleParams: (route) => ({
            preset: String(route.params.preset),
          }),
        },
        component: () => import('@/pages/overlays/AROverlay.vue'),
      },
      {
        path: 'info',
        name: 'overlayInfo',
        meta: {
          titleKey: 'overlays.info',
        },
        component: () => import('@/pages/overlays/InfoOverlay.vue'),
      },
      {
        path: 'lane/:number',
        name: 'overlayLane',
        meta: {
          titleKey: 'overlays.lane',
          titleParams: (route) => ({
            number: String(route.params.number),
          }),
        },
        component: () => import('@/pages/overlays/LaneOverlay.vue'),
      },
      {
        path: 'splits',
        name: 'overlaySplits',
        meta: {
          titleKey: 'overlays.splits',
        },
        component: () => import('@/pages/overlays/SplitsOverlay.vue'),
      },
      {
        path: 'startlist',
        name: 'overlayStartList',
        meta: {
          titleKey: 'overlays.startList',
        },
        component: () => import('@/pages/overlays/StartListOverlay.vue'),
      },
      {
        path: 'timer',
        name: 'overlayTimer',
        meta: {
          titleKey: 'overlays.timer',
        },
        component: () => import('@/pages/overlays/TimerOverlay.vue'),
      },
    ],
  },
];

/**
 * overlayRouter only knows the overlay pages. An unknown path renders nothing,
 * since no fallback page makes sense in a video.
 */
export const overlayRouter = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});
