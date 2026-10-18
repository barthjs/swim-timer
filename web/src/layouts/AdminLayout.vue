<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';
import type { NavigationMenuItem } from '@nuxt/ui';

import { useMutation } from '@/composables/form/useMutation';
import AppVersion from '@/components/system/AppVersion.vue';
import { useAuthStore } from '@/stores/auth';
import { useMetaStore } from '@/stores/meta';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();

const authStore = useAuthStore();
const metaStore = useMetaStore();

// Pages such as the editor use the whole width of the main area instead of the container width,
// and from lg up fill the height between header and footer instead of scrolling.
const fullWidth = computed(() => !!route.meta.fullWidth);
const fillViewport = computed(() => {
  const fill = route.meta.fillViewport;
  return typeof fill === 'function' ? fill(route) : !!fill;
});

const appName = computed(() => metaStore.meta?.name || 'Swim-Timer');

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
    label: t('admin.nav.editor'),
    icon: 'i-lucide-scan',
    to: {
      name: 'editor',
    },
  },
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

const { mutate: logout, pending: loggingOut } = useMutation(() => authStore.logout(), {
  onSuccess: () => {
    void router.push({
      name: 'dashboard',
    });
  },
});
</script>

<template>
  <!--
  The footer shares the viewport height with the header and the main area,
  so the version stays visible on short pages instead of being pushed below the fold.
  A page that fills the viewport gets exactly the height left between header and footer.
  -->
  <div class="flex min-h-dvh flex-col" :class="{ 'lg:h-dvh': fillViewport }">
    <UHeader>
      <template #title>
        <RouterLink
          :to="{ name: 'dashboard' }"
          class="text-highlighted flex items-center gap-2 font-semibold"
        >
          <img src="@/assets/images/logo/favicon.svg" alt="" class="size-6" />
          {{ appName }}
        </RouterLink>
      </template>

      <UNavigationMenu :items="navItems" />

      <template #right>
        <UTooltip :text="t('admin.openScoreboard')">
          <UButton
            icon="i-lucide-external-link"
            color="neutral"
            variant="ghost"
            :aria-label="t('admin.openScoreboard')"
            :to="{ name: 'scoreboard' }"
            target="_blank"
          />
        </UTooltip>
        <UColorModeSwitch />
        <UButton
          v-if="authStore.authenticated"
          :label="t('admin.logout')"
          icon="i-lucide-log-out"
          color="neutral"
          variant="ghost"
          :loading="loggingOut"
          @click="logout()"
        />
        <UButton
          v-else
          :label="t('admin.login')"
          icon="i-lucide-log-in"
          color="neutral"
          variant="ghost"
          :to="{ name: 'login' }"
        />
      </template>

      <template #body>
        <UNavigationMenu :items="navItems" orientation="vertical" class="-mx-2.5" />
      </template>
    </UHeader>

    <UMain class="min-h-0 flex-1" :class="{ 'lg:flex lg:flex-col': fillViewport }">
      <!-- A full width page keeps the side padding of the container without its maximum width. -->
      <div
        v-if="fullWidth"
        class="w-full px-4 pt-6 pb-2 sm:px-6 lg:px-8"
        :class="{ 'lg:flex lg:min-h-0 lg:flex-1 lg:flex-col': fillViewport }"
      >
        <RouterView />
      </div>
      <UContainer
        v-else
        class="pt-6 pb-2"
        :class="{ 'lg:flex lg:min-h-0 lg:flex-1 lg:flex-col': fillViewport }"
      >
        <RouterView />
      </UContainer>
    </UMain>

    <UFooter :ui="{ container: 'py-1 lg:py-1', center: 'mt-0' }">
      <AppVersion class="text-muted" />
    </UFooter>
  </div>
</template>
