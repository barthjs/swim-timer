<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useTimeoutFn } from '@vueuse/core';

import { metaRepository } from '@/api/repositories/meta';
import { updateRepository } from '@/api/repositories/update';
import type { UpdateCheckResponse, UpdateStatusResponse } from '@/api/types';
import HelpHint from '@/components/form/HelpHint.vue';
import { useApiError } from '@/composables/useApiError';
import { useConfirm } from '@/composables/ui/useConfirm';
import { useMutation } from '@/composables/form/useMutation';
import { RestartState, isRestartAnswer, restartState } from '@/lib/update';

// How long to wait after an answer before the update status and, during the restart, the server are asked again.
const POLL_INTERVAL_MS = 2000;

const emit = defineEmits<{
  // Reports whether the update is starting, running, or restarting, so the dialog stays open until the reload
  busy: [busy: boolean];
}>();

const { t } = useI18n();
const { confirm } = useConfirm();
const { messageFor } = useApiError();

const check = ref<UpdateCheckResponse | null>(null);
const status = ref<UpdateStatusResponse | null>(null);
// The server restarts after the update is installed and does not answer for a moment.
const restarting = ref(false);
// The version before the update and the time the restart began, to tell when the new version answers.
const previousVersion = ref('');
const restartStartedAt = ref(0);
// Why following the update stopped: an error answer of the server, or a restart that took too long.
const pollError = ref<string | null>(null);
// Whether the update is followed, a request that is still running when it stops schedules no further one
let following = false;

const { mutate: runCheck, pending: checking } = useMutation(() => updateRepository.check(), {
  onSuccess: (result) => {
    check.value = result;
  },
});

const { mutate: start, pending: starting } = useMutation(() => updateRepository.start(), {
  onSuccess: () => {
    previousVersion.value = check.value?.current ?? '';
    pollError.value = null;
    status.value = {
      state: 'running',
    };
    following = true;
    schedulePoll();
  },
});

// The next request is only scheduled once the previous one finished, so the requests do not pile up
// while the server does not answer.
const { start: schedulePoll, stop: cancelPoll } = useTimeoutFn(
  () => void poll(),
  POLL_INTERVAL_MS,
  {
    immediate: false,
  },
);

/**
 * poll follows the update until it is installed, then waits until the restarted server reports the new version.
 * It schedules the next request as long as the update is followed.
 */
async function poll(): Promise<void> {
  if (restarting.value) {
    await waitForRestart();
  } else {
    await followUpdate();
  }
  if (following) schedulePoll();
}

/** followUpdate reads the state of the update and switches to the restart once it is installed. */
async function followUpdate(): Promise<void> {
  try {
    status.value = await updateRepository.getStatus();
    if (status.value.state === 'applied') beginRestart();
    if (status.value.state === 'failed') stopFollowing();
  } catch (err) {
    // An unavailable server or a forgotten session means a restart, any other error ends following the update
    if (isRestartAnswer(err)) {
      beginRestart();
      return;
    }
    stopFollowing();
    pollError.value = messageFor(err);
  }
}

/** stopFollowing ends following the update. */
function stopFollowing(): void {
  following = false;
  cancelPoll();
}

/** beginRestart switches to waiting for the restarted server. */
function beginRestart(): void {
  restarting.value = true;
  restartStartedAt.value = Date.now();
}

/**
 * waitForRestart reloads the page once the server answers with another version than before the update,
 * and gives up after a timeout. The installed release can be newer than the checked one.
 */
async function waitForRestart(): Promise<void> {
  let reported: string | null = null;
  try {
    const meta = await metaRepository.get();
    reported = meta.version;
  } catch {
    // The server is not back yet
  }

  const state = restartState(previousVersion.value, reported, restartStartedAt.value, Date.now());
  if (state === RestartState.Restarted) {
    stopFollowing();
    window.location.reload();
    return;
  }
  if (state === RestartState.TimedOut) {
    stopFollowing();
    restarting.value = false;
    pollError.value = t('update.restartTimeout');
  }
}

/** install starts the update after a confirmation. */
async function install(): Promise<void> {
  if (!check.value) return;
  const confirmed = await confirm({
    title: t('update.confirmTitle', {
      version: check.value.latest,
    }),
    description: t('update.confirmDescription'),
    confirmLabel: t('update.install'),
    icon: 'i-lucide-download',
  });
  if (confirmed) await start();
}

onMounted(() => void runCheck());
onBeforeUnmount(stopFollowing);

const busy = computed(
  () => !pollError.value && (status.value?.state === 'running' || restarting.value),
);

// Closing the panel stops following the update, so the page would not reload into the new version
watch(
  () => starting.value || busy.value,
  (value) => emit('busy', value),
  {
    immediate: true,
  },
);
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center gap-3">
      <UButton
        icon="i-lucide-refresh-cw"
        color="neutral"
        variant="subtle"
        :loading="checking"
        :disabled="busy"
        @click="runCheck()"
      >
        {{ t('update.check') }}
      </UButton>
      <HelpHint :text="t('update.hint')" />
      <span v-if="check" class="text-muted text-sm">
        {{ t('update.current', { version: check.current }) }}
      </span>
    </div>

    <template v-if="check">
      <UAlert
        v-if="!check.newer"
        color="success"
        variant="subtle"
        icon="i-lucide-circle-check"
        :title="t('update.latest')"
      />

      <UAlert
        v-else
        color="info"
        variant="subtle"
        icon="i-lucide-arrow-up-circle"
        :title="t('update.available', { version: check.latest })"
        :description="check.unsupported ? t(`update.unsupported.${check.unsupported}`) : undefined"
      >
        <template #actions>
          <UButton
            :to="check.url"
            target="_blank"
            icon="i-lucide-external-link"
            color="neutral"
            variant="outline"
          >
            {{ t('update.releaseNotes') }}
          </UButton>
          <UButton
            v-if="!check.unsupported"
            icon="i-lucide-download"
            :loading="starting || busy"
            :disabled="busy"
            @click="install"
          >
            {{ t('update.install') }}
          </UButton>
        </template>
      </UAlert>
    </template>

    <UAlert
      v-if="restarting"
      color="neutral"
      variant="subtle"
      icon="i-lucide-loader-circle"
      :title="t('update.restarting')"
    />
    <UAlert
      v-else-if="pollError"
      color="error"
      variant="subtle"
      icon="i-lucide-circle-x"
      :title="t('update.failed')"
      :description="pollError"
    />
    <UAlert
      v-else-if="status?.state === 'running'"
      color="neutral"
      variant="subtle"
      icon="i-lucide-loader-circle"
      :title="t('update.installing')"
    />
    <UAlert
      v-else-if="status?.state === 'failed'"
      color="error"
      variant="subtle"
      icon="i-lucide-circle-x"
      :title="t('update.failed')"
      :description="status.error"
    />
  </div>
</template>
