<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import type { NavigationMenuItem } from '@nuxt/ui';

import { useMutation } from '@/composables/form/useMutation';
import DashboardFrame from '@/components/dashboard/DashboardFrame.vue';
import AppVersion from '@/components/system/AppVersion.vue';
import FooterLinks, { type FooterLink } from '@/components/system/FooterLinks.vue';
import { docsUrl } from '@/lib/app/docs';
import { loginLocation } from '@/lib/auth/redirect';
import { useAuthStore } from '@/stores/auth';
import { useMetaStore } from '@/stores/meta';

const REPOSITORY_URL = 'https://github.com/barthjs/swim-timer';

const { t } = useI18n();

// The icon links of the footer of the sidebar
const footerLinks = computed<FooterLink[]>(() => [
  {
    label: t('common.docs'),
    icon: 'i-lucide-book-open',
    to: docsUrl(),
    external: true,
  },
  {
    label: t('common.repository'),
    icon: 'i-lucide-github',
    to: REPOSITORY_URL,
    external: true,
  },
]);
const route = useRoute();
const router = useRouter();

const authStore = useAuthStore();
const metaStore = useMetaStore();

// From lg up, a page that fills the viewport keeps its height inside the panel, so its table scrolls on its own
const fillViewport = computed(() => {
  const fill = route.meta.fillViewport;
  return typeof fill === 'function' ? fill(route) : !!fill;
});

const appName = computed(() => metaStore.meta?.name || 'Swim-Timer');

// The login returns to the page it was opened from
const loginTarget = computed(() =>
  loginLocation(
    {
      name: 'login',
    },
    route.fullPath,
  ),
);

const navItems = computed<NavigationMenuItem[]>(() => [
  {
    label: t('admin.nav.dashboard'),
    icon: 'i-lucide-layout-dashboard',
    to: {
      name: 'dashboard',
    },
    exact: true,
  },
  {
    label: t('admin.nav.meet'),
    icon: 'i-lucide-calendar-range',
    to: {
      name: 'meet',
    },
  },
  {
    label: t('admin.nav.meetData'),
    icon: 'i-lucide-file-spreadsheet',
    to: {
      name: 'meetData',
    },
  },
  {
    label: t('admin.nav.records'),
    icon: 'i-lucide-trophy',
    to: {
      name: 'records',
    },
  },
  {
    label: t('admin.nav.editor'),
    icon: 'i-lucide-scan',
    to: {
      name: 'editor',
    },
  },
]);

// The configuration sits at the bottom of the sidebar, apart from the pages of the meet
const bottomNavItems = computed<NavigationMenuItem[]>(() => [
  {
    label: t('admin.nav.config'),
    icon: 'i-lucide-settings',
    to: {
      name: 'config',
    },
  },
]);

// The dashboard is public, so the header offers a login or a logout depending on the session
onMounted(() => {
  void authStore.ensureLoaded();
});

const { mutate: logout, pending: loggingOut } = useMutation(async () => authStore.logout(), {
  onSuccess: () => {
    void router.push({
      name: 'dashboard',
    });
  },
});
</script>

<template>
  <DashboardFrame
    :items="navItems"
    :bottom-items="bottomNavItems"
    :home-to="{
      name: 'dashboard',
    }"
    storage-key="admin"
    :desktop-navbar="false"
  >
    <template #logo>
      <span class="text-highlighted flex items-center gap-2 font-semibold">
        <img src="@/assets/images/logo/favicon.svg" alt="" class="size-6" />
        {{ appName }}
      </span>
    </template>

    <template #footer="{ collapsed, close }">
      <div class="flex w-full flex-col gap-2">
        <div class="flex items-center gap-1" :class="collapsed ? 'flex-col' : ''">
          <UTooltip
            :text="authStore.authenticated ? t('admin.logout') : t('admin.login')"
            :disabled="!collapsed"
          >
            <UButton
              v-if="authStore.authenticated"
              :label="collapsed ? undefined : t('admin.logout')"
              :aria-label="t('admin.logout')"
              icon="i-lucide-log-out"
              color="neutral"
              variant="ghost"
              :loading="loggingOut"
              @click="logout()"
            />
            <UButton
              v-else
              :label="collapsed ? undefined : t('admin.login')"
              :aria-label="t('admin.login')"
              icon="i-lucide-log-in"
              color="neutral"
              variant="ghost"
              :to="loginTarget"
              @click="close"
            />
          </UTooltip>
          <UTooltip :text="t('admin.openScoreboard')">
            <UButton
              icon="i-lucide-external-link"
              color="neutral"
              variant="ghost"
              :aria-label="t('admin.openScoreboard')"
              :to="{ name: 'scoreboard' }"
              target="_blank"
              :class="collapsed ? '' : 'ms-auto'"
            />
          </UTooltip>
          <UColorModeSwitch />
        </div>
        <div class="flex w-full items-center" :class="collapsed ? 'flex-col' : 'justify-between'">
          <AppVersion v-if="!collapsed" :repository-url="REPOSITORY_URL">
            <p class="text-muted text-sm">{{ t('admin.updateOverlays') }}</p>
          </AppVersion>
          <FooterLinks :links="footerLinks" :class="collapsed ? 'flex-col' : ''" />
        </div>
      </div>
    </template>

    <div class="flex flex-col" :class="{ 'lg:min-h-0 lg:flex-1': fillViewport }">
      <RouterView />
    </div>
  </DashboardFrame>
</template>
