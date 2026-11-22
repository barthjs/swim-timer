<script setup lang="ts">
import { computed, ref } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { NavigationMenuItem } from '@nuxt/ui';

// The frame of an app area with a sidebar for the navigation and a panel for the page that scrolls inside the window.
// The navbar holds the global actions, and the sidebar remembers its width and state in the browser.
const props = withDefaults(
  defineProps<{
    items: NavigationMenuItem[];
    // The pages at the bottom of the sidebar, apart from the others
    bottomItems?: NavigationMenuItem[];
    // The link of the logo in the header of the sidebar
    homeTo: RouteLocationRaw;
    // Names the sizes and the state of the sidebar in the browser storage
    storageKey: string;
    // Shows the navbar from lg up as well. Without it, only small screens get a navbar with the logo
    // that opens the sidebar, and the global actions belong in the footer of the sidebar.
    desktopNavbar?: boolean;
  }>(),
  {
    bottomItems: () => [],
    desktopNavbar: true,
  },
);

defineSlots<{
  logo: () => unknown;
  // The controls after the toggle of the sidebar
  leading: () => unknown;
  // The middle of the navbar
  center: () => unknown;
  actions: () => unknown;
  footer: (props: { collapsed: boolean; close: () => void }) => unknown;
  default: () => unknown;
}>();

// The sidebar is a slideover below the lg breakpoint, which closes once a page is chosen
const open = ref(false);

/** Closes the slideover of the sidebar. */
function close() {
  open.value = false;
}

/** Closes the slideover once an item is chosen. */
const withClose = (items: NavigationMenuItem[]): NavigationMenuItem[] =>
  items.map((item) => ({
    ...item,
    onSelect: close,
  }));

const navigationItems = computed(() => withClose(props.items));
const bottomNavigationItems = computed(() => withClose(props.bottomItems));
</script>

<template>
  <UDashboardGroup storage="local" :storage-key="storageKey" unit="rem">
    <UDashboardSidebar
      :id="storageKey"
      v-model:open="open"
      collapsible
      resizable
      :default-size="15"
      :min-size="12"
      :max-size="22"
      :ui="{
        root: 'bg-chrome',
        footer: 'lg:border-t lg:border-default',
      }"
    >
      <!-- The toggle keeps its place in both states, so it does not jump when the sidebar collapses -->
      <template #header="{ collapsed }">
        <div class="flex w-full items-center gap-1" :class="collapsed ? 'justify-center' : ''">
          <ULink v-if="!collapsed" :to="homeTo" class="min-w-0 flex-1">
            <slot name="logo" />
          </ULink>
          <UDashboardSidebarCollapse />
        </div>
      </template>

      <template #default="{ collapsed }">
        <UNavigationMenu
          :items="navigationItems"
          :collapsed="collapsed"
          orientation="vertical"
          tooltip
          popover
        />
        <UNavigationMenu
          v-if="bottomNavigationItems.length > 0"
          :items="bottomNavigationItems"
          :collapsed="collapsed"
          orientation="vertical"
          tooltip
          class="mt-auto"
        />
      </template>

      <template #footer="{ collapsed }">
        <slot name="footer" :collapsed="collapsed ?? false" :close="close" />
      </template>
    </UDashboardSidebar>

    <UDashboardPanel :id="`${storageKey}-page`" :ui="{ body: 'gap-3 p-3 sm:gap-4 sm:p-4' }">
      <template #header>
        <UDashboardNavbar
          :ui="{
            root: desktopNavbar ? '' : 'lg:hidden',
            center: 'flex min-w-0 flex-1 justify-center',
          }"
        >
          <template #toggle>
            <UDashboardSidebarToggle />
            <ULink v-if="!desktopNavbar" :to="homeTo" class="min-w-0">
              <slot name="logo" />
            </ULink>
            <slot name="leading" />
          </template>

          <slot name="center" />

          <template #right>
            <slot name="actions" />
          </template>
        </UDashboardNavbar>
      </template>

      <template #body>
        <slot />
      </template>
    </UDashboardPanel>
  </UDashboardGroup>
</template>
