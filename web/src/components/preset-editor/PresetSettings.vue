<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { TextAnchor } from '@/api/types';
import EditorPanel from '@/components/editor/EditorPanel.vue';
import HelpHint from '@/components/form/HelpHint.vue';
import { usePresetForm } from '@/composables/usePresetForm';
import { useRatioLock } from '@/composables/useRatioLock';
import { aspectRatioDiffers } from '@/lib/editor/geometry';
import {
  FRAME_ASPECTS,
  closestResolution,
  findAspect,
  type FrameAspect,
  type FrameSize,
} from '@/lib/frame';
import {
  DIMENSION_MAX,
  DIMENSION_MIN,
  PRESET_COMMENT_MAX,
  PRESET_NAME_MAX,
  TEXT_SIZE_MAX,
  TEXT_SIZE_MIN,
  frameOf,
} from '@/lib/lanes';
import { countLabel } from '@/lib/form/validation';

type LaneOrder = 'normal' | 'reversed';

// The selection of the aspect ratio: a common one, or any ratio the fields below give.
type AspectChoice = FrameAspect['key'] | 'free';

// The settings that apply to all lanes of the open preset, with undo and redo.
// The fields belong to the editor page's form, which shows the errors of a failed save.
const { t } = useI18n();
const presetForm = usePresetForm();
const { preset, dirtyForm } = presetForm;

// While the ratio is locked, the other dimension of the resolution follows a changed one. lockedAspect is
// the common aspect ratio the lock was set to. The selection keeps showing it, although sizes rounded to whole pixels
// do not match it exactly.
const ratioLock = useRatioLock();
const lockedAspect = ref<FrameAspect['key'] | null>(null);

const frame = computed<FrameSize>(() => frameOf(preset));

// An opened preset with a common aspect ratio starts locked to it. The count of opened presets
// also catches a new draft that follows another draft, whose saved name stays empty.
watch(
  () => presetForm.openCount.value,
  () => lockToFrame(),
  {
    immediate: true,
  },
);

// Undo, redo, or a typed size can give the frame another aspect ratio.
// The lock then follows the common aspect ratio of the new frame, or ends for any other ratio.
watch(frame, (next) => {
  if (!ratioLock.locked.value) return;
  if (
    !aspectRatioDiffers(next, {
      width: ratioLock.ratio.value,
      height: 1,
    })
  )
    return;
  lockToFrame();
});

const laneOrder = computed<LaneOrder>({
  get: () => (preset.reverseLanes ? 'reversed' : 'normal'),
  set: (value) => {
    preset.reverseLanes = value === 'reversed';
    presetForm.editor.history.commit();
  },
});

const laneOrderItems = computed(() => [
  {
    label: t('pages.editor.topLaneOne'),
    value: 'normal',
  },
  {
    label: t('pages.editor.topHighestLane'),
    value: 'reversed',
  },
]);

// The selection shows the locked aspect ratio. An unlocked ratio, or one locked at an uncommon value, is free.
const aspectChoice = computed<AspectChoice>({
  get: () => (ratioLock.locked.value && lockedAspect.value ? lockedAspect.value : 'free'),
  set: (key) => {
    const aspect = FRAME_ASPECTS.find((item) => item.key === key);
    if (!aspect) {
      unlockRatio();
      return;
    }
    lockAspect(aspect);
    if (findAspect(frame.value) === aspect) return;
    const resolution = closestResolution(aspect, frame.value);
    presetForm.setFrameSize(resolution.width, resolution.height);
  },
});

// The common aspect ratios first, then the free ratio.
const aspectItems = computed(() => [
  ...FRAME_ASPECTS.map((aspect) => ({
    label: aspect.key,
    value: aspect.key,
  })),
  {
    label: t('pages.editor.aspectFree'),
    value: 'free',
  },
]);

// The common resolutions of the selected aspect ratio, or of the ratio the free size has.
const resolutions = computed<readonly FrameSize[]>(() => {
  const selected = FRAME_ASPECTS.find((aspect) => aspect.key === aspectChoice.value);
  return (selected ?? findAspect(frame.value))?.resolutions ?? [];
});

// The sides of a lane the text can start at.
const textAnchorItems = computed(() => [
  {
    label: t('pages.editor.anchorLeft'),
    value: TextAnchor.Left,
  },
  {
    label: t('pages.editor.anchorRight'),
    value: TextAnchor.Right,
  },
]);

/** lockToFrame locks the ratio at the common aspect ratio of the frame, or unlocks it for any other. */
function lockToFrame(): void {
  const aspect = findAspect(frame.value);
  if (aspect) {
    lockAspect(aspect);
    return;
  }
  unlockRatio();
}

/** lockAspect locks the ratio of the resolution at a common aspect ratio. */
function lockAspect(aspect: FrameAspect): void {
  lockedAspect.value = aspect.key;
  ratioLock.lock(aspect.width / aspect.height);
}

/** toggleRatioLock locks the ratio of the resolution at its current value, or unlocks it. */
function toggleRatioLock(): void {
  if (ratioLock.locked.value) {
    unlockRatio();
    return;
  }
  lockedAspect.value = findAspect(frame.value)?.key ?? null;
  ratioLock.toggle(preset.frameWidth / preset.frameHeight);
}

/** unlockRatio lets the width and the height of the resolution change independently. */
function unlockRatio(): void {
  ratioLock.unlock();
  lockedAspect.value = null;
}

/** onFrameWidth changes the width of the reference resolution, keeping the lanes in place. */
function onFrameWidth(width: number | null | undefined): void {
  if (typeof width !== 'number') return;
  const height = ratioLock.locked.value
    ? clampDimension(ratioLock.heightFor(width))
    : preset.frameHeight;
  presetForm.setFrameSize(width, height);
}

/** onFrameHeight changes the height of the reference resolution, keeping the lanes in place. */
function onFrameHeight(height: number | null | undefined): void {
  if (typeof height !== 'number') return;
  const width = ratioLock.locked.value
    ? clampDimension(ratioLock.widthFor(height))
    : preset.frameWidth;
  presetForm.setFrameSize(width, height);
}

/** clampDimension rounds a derived dimension to whole pixels in the range the backend accepts. */
function clampDimension(value: number): number {
  return Math.min(DIMENSION_MAX, Math.max(DIMENSION_MIN, Math.round(value)));
}

/** isCurrentResolution reports whether the preset has the given resolution. */
function isCurrentResolution(resolution: FrameSize): boolean {
  return resolution.width === preset.frameWidth && resolution.height === preset.frameHeight;
}
</script>

<template>
  <EditorPanel :title="t('pages.editor.preset')" storage-key="editor.panel.preset">
    <!-- The name is a field below, so the header only marks an unsaved draft and unsaved changes. -->
    <template #summary>
      <UBadge
        v-if="presetForm.isDraft.value"
        :label="t('pages.editor.draft')"
        color="neutral"
        variant="soft"
      />
      <UBadge
        v-if="presetForm.isDirty.value"
        :label="t('pages.editor.changed')"
        color="warning"
        variant="solid"
      />
    </template>

    <UFormField
      :label="t('fields.name')"
      :hint="countLabel(preset.name, PRESET_NAME_MAX)"
      name="name"
      required
    >
      <UInput
        v-model="preset.name"
        v-bind="dirtyForm.fieldProps('name')"
        :maxlength="PRESET_NAME_MAX"
        class="w-full"
      />
    </UFormField>

    <UFormField
      :label="t('fields.comment')"
      :hint="countLabel(preset.comment ?? '', PRESET_COMMENT_MAX)"
      name="comment"
    >
      <UInput
        :model-value="preset.comment ?? ''"
        v-bind="dirtyForm.fieldProps('comment')"
        :maxlength="PRESET_COMMENT_MAX"
        class="w-full"
        @update:model-value="(value: string) => (preset.comment = value || null)"
        @change="() => presetForm.editor.history.commit()"
      />
    </UFormField>

    <div class="flex flex-col gap-2">
      <UFormField :label="t('pages.editor.aspectRatio')" name="aspectRatio">
        <template #hint>
          <HelpHint :text="t('pages.editor.help.frameSize')" />
        </template>
        <USelect v-model="aspectChoice" :items="aspectItems" class="w-full" />
      </UFormField>
      <div class="flex items-end gap-2">
        <UFormField
          :label="t('fields.frameWidth')"
          name="frameWidth"
          required
          class="min-w-0 flex-1"
        >
          <UInputNumber
            orientation="vertical"
            :model-value="preset.frameWidth"
            v-bind="dirtyForm.fieldProps('frameWidth')"
            :min="DIMENSION_MIN"
            :max="DIMENSION_MAX"
            class="w-full"
            @update:model-value="onFrameWidth"
          />
        </UFormField>
        <UTooltip
          :text="
            ratioLock.locked.value ? t('pages.editor.unlockRatio') : t('pages.editor.lockRatio')
          "
        >
          <UButton
            :icon="ratioLock.locked.value ? 'i-lucide-link' : 'i-lucide-link-2-off'"
            :color="ratioLock.locked.value ? 'primary' : 'neutral'"
            variant="soft"
            :aria-label="
              ratioLock.locked.value ? t('pages.editor.unlockRatio') : t('pages.editor.lockRatio')
            "
            :aria-pressed="ratioLock.locked.value"
            @click="toggleRatioLock"
          />
        </UTooltip>
        <UFormField
          :label="t('fields.frameHeight')"
          name="frameHeight"
          required
          class="min-w-0 flex-1"
        >
          <UInputNumber
            orientation="vertical"
            :model-value="preset.frameHeight"
            v-bind="dirtyForm.fieldProps('frameHeight')"
            :min="DIMENSION_MIN"
            :max="DIMENSION_MAX"
            class="w-full"
            @update:model-value="onFrameHeight"
          />
        </UFormField>
      </div>
      <div v-if="resolutions.length" class="grid grid-cols-2 gap-1 @sm:grid-cols-4">
        <UButton
          v-for="resolution in resolutions"
          :key="`${resolution.width}x${resolution.height}`"
          :label="`${resolution.width}×${resolution.height}`"
          size="xs"
          class="justify-center px-1.5 tabular-nums"
          :color="isCurrentResolution(resolution) ? 'primary' : 'neutral'"
          :variant="isCurrentResolution(resolution) ? 'soft' : 'outline'"
          @click="presetForm.setFrameSize(resolution.width, resolution.height)"
        />
      </div>
    </div>

    <UFormField :label="t('fields.textSize')" name="textSize" required>
      <template #hint>
        <HelpHint :text="t('pages.editor.help.textSize')" />
      </template>
      <UInputNumber
        v-model="preset.textSize"
        v-bind="dirtyForm.fieldProps('textSize')"
        :min="TEXT_SIZE_MIN"
        :max="TEXT_SIZE_MAX"
        class="w-full"
        @update:model-value="() => presetForm.editor.history.commit()"
      />
    </UFormField>

    <UFormField :label="t('fields.reverseLanes')" name="reverseLanes">
      <template #hint>
        <HelpHint :text="t('pages.editor.help.laneOrder')" />
      </template>
      <USelect
        v-model="laneOrder"
        v-bind="dirtyForm.fieldProps('reverseLanes')"
        :items="laneOrderItems"
        class="w-full"
      />
    </UFormField>

    <UFormField :label="t('fields.textAnchor')" name="textAnchor" required>
      <template #hint>
        <HelpHint :text="t('pages.editor.help.textAnchor')" />
      </template>
      <USelect
        v-model="preset.textAnchor"
        v-bind="dirtyForm.fieldProps('textAnchor')"
        :items="textAnchorItems"
        class="w-full"
        @update:model-value="() => presetForm.editor.history.commit()"
      />
    </UFormField>
  </EditorPanel>
</template>
