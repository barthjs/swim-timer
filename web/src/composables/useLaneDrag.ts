import { ref, toRaw, toValue, type MaybeRefOrGetter } from 'vue';

import type { CornerKey } from '@/api/types';
import { isAdditiveSelect } from '@/composables/editor/useEditorSelection';
import { usePointerDrag } from '@/composables/usePointerDrag';
import type { PresetForm } from '@/composables/usePresetForm';
import {
  constrainToAxis,
  translateBounds,
  unionBounds,
  type Bounds,
  type Point,
} from '@/lib/editor/geometry';
import {
  SNAP_DISTANCE,
  boundsTargets,
  mergeTargets,
  noSnap,
  snapBounds,
  snapOffset,
  snapPoint,
  snapPointToAxes,
  snapValue,
  type AxisTargets,
  type PointTargets,
  type SnapFeedback,
} from '@/lib/editor/snap';
import { measureGaps, spacingTargets } from '@/lib/editor/spacing';
import {
  CORNERS,
  clampCoordinate,
  frameOf,
  laneBounds,
  laneBoundsExcluding,
  lanePointTargets,
  translateLane,
  type PixelLane,
} from '@/lib/lanes';

type SnapTargets = {
  points: PointTargets;
  axes: AxisTargets;
  // The bounds of the lanes that stay, which the distances are measured to.
  others: Bounds[];
};

type DragState =
  | {
      mode: 'corner';
      laneKey: string;
      corner: CornerKey;
      start: Point;
      targets: SnapTargets;
    }
  | {
      mode: 'lane';
      initialLanes: Record<string, PixelLane>;
      start: Point;
      targets: SnapTargets;
    };

export type DragMode = DragState['mode'];

export type LaneDragOptions = {
  snapping: MaybeRefOrGetter<boolean>;
  // Converts a position in the browser window into pixels of the reference resolution.
  toFrame: (clientX: number, clientY: number) => Point;
  // The screen pixels per pixel of the reference resolution, so the snap reach stays the same on screen.
  scale: MaybeRefOrGetter<number>;
  // The guides the lanes snap to, in pixels of the reference resolution.
  guides: () => AxisTargets;
};

/**
 * useLaneDrag moves lanes and lane corners with the pointer, in pixels of the reference resolution. A gesture records
 * one history entry when it ends or the browser cancels it. With snapping on, dragged corners snap onto the corners
 * and edges of the other lanes and the image border. Without such a target in reach, edges and centers align
 * with those of the other lanes, of the image, and with the guides, and moved lanes keep even gaps to their
 * neighbors. Holding Ctrl or Cmd suspends snapping, and Shift keeps a drag on one axis.
 */
export function useLaneDrag(options: LaneDragOptions, presetForm: PresetForm) {
  const mode = ref<DragMode | null>(null);
  // What the dragged corner or lanes snapped to and their distances, shown on the canvas
  const feedback = ref<SnapFeedback>(noSnap());
  let state: DragState | null = null;

  const pointerDrag = usePointerDrag({
    onMove: (_dx, _dy, event) => moveTo(event),
    onEnd: () => {
      mode.value = null;
      feedback.value = noSnap();
      state = null;
      presetForm.editor.dragging.value = false;
      presetForm.editor.history.commit();
    },
  });

  /**
   * targetsExcluding collects the snap targets of the guides and of every lane except the dragged ones
   * and the hidden ones.
   */
  function targetsExcluding(dragged: ReadonlySet<string>): SnapTargets {
    const keys = new Set([...dragged, ...presetForm.editor.visibility.hidden.value]);
    const frame = frameOf(presetForm.preset);
    const others = laneBoundsExcluding(presetForm.preset.lanes, keys);
    return {
      points: lanePointTargets(presetForm.preset.lanes, keys, frame),
      axes: mergeTargets(boundsTargets(others, frame), options.guides()),
      others,
    };
  }

  /** snapEnabled reports whether snapping applies, since holding Ctrl or Cmd turns it off. */
  function snapEnabled(event: PointerEvent): boolean {
    return toValue(options.snapping) && !event.ctrlKey && !event.metaKey;
  }

  /** snapDistance returns the snap reach in pixels of the reference resolution. */
  function snapDistance(): number {
    const scale = toValue(options.scale);
    return scale > 0 ? SNAP_DISTANCE / scale : SNAP_DISTANCE;
  }

  /**
   * moveTo moves a corner or the lanes by the distance the pointer traveled since the start of the gesture,
   * along one axis only while Shift is held. Lanes move from their position at the start, so clamping a lane
   * at the edge of the image does not change the distances within the selection.
   */
  function moveTo(event: PointerEvent): void {
    if (!state) return;

    const position = options.toFrame(event.clientX, event.clientY);
    const traveled = {
      x: position.x - state.start.x,
      y: position.y - state.start.y,
    };
    const offset = event.shiftKey ? constrainToAxis(traveled) : traveled;
    if (state.mode === 'corner') {
      moveCorner(state, offset, event);
      return;
    }
    moveLanes(state.initialLanes, state.targets, offset, event);
  }

  /** moveCorner moves a corner of a lane from where the gesture started, snapped if enabled. */
  function moveCorner(
    drag: Extract<DragState, { mode: 'corner' }>,
    offset: Point,
    event: PointerEvent,
  ): void {
    const lane = presetForm.preset.lanes[drag.laneKey];
    if (!lane) return;

    let point = {
      x: drag.start.x + offset.x,
      y: drag.start.y + offset.y,
    };
    feedback.value = noSnap();
    if (snapEnabled(event)) {
      const snapped = snapPoint(point, drag.targets.points, snapDistance());
      if (snapped.target && !event.shiftKey) {
        point = snapped.point;
        feedback.value = {
          point: snapped.target,
          lines: [],
          gaps: [],
        };
      } else {
        const aligned = snapPointToAxes(point, drag.targets.axes, snapDistance());
        // A corner kept on one axis does not leave it for a snap
        point = {
          x: event.shiftKey && offset.x === 0 ? point.x : aligned.point.x,
          y: event.shiftKey && offset.y === 0 ? point.y : aligned.point.y,
        };
        feedback.value = aligned.feedback;
      }
    }

    lane[drag.corner].x = clampCoordinate(point.x, presetForm.preset.frameWidth);
    lane[drag.corner].y = clampCoordinate(point.y, presetForm.preset.frameHeight);
  }

  /**
   * moveLanes moves the dragged lanes by the pointer distance, snapped if enabled, and shows their distances
   * to the neighbors. Every corner stays inside the image, so a lane pushed against the border gets narrower.
   */
  function moveLanes(
    initialLanes: Record<string, PixelLane>,
    targets: SnapTargets,
    pointerShift: Point,
    event: PointerEvent,
  ): void {
    const moving: Array<[PixelLane, PixelLane]> = [];
    for (const key of presetForm.editor.selection.selected.value) {
      const lane = presetForm.preset.lanes[key];
      const initial = initialLanes[key];
      if (lane && initial && !lane.locked) {
        moving.push([lane, initial]);
      }
    }
    const initialBounds = unionBounds(moving.map(([, initial]) => laneBounds(initial)));
    if (!initialBounds) return;

    let offset = pointerShift;
    feedback.value = noSnap();
    if (snapEnabled(event)) {
      offset = snapLanes(moving, initialBounds, targets, offset, event.shiftKey);
    }

    const frame = frameOf(presetForm.preset);
    for (const [lane, initial] of moving) {
      translateLane(lane, initial, offset.x, offset.y, frame);
    }
    feedback.value = {
      ...feedback.value,
      gaps: measureGaps(translateBounds(initialBounds, offset), targets.others, frame),
    };
  }

  /**
   * snapLanes snaps the closest corner of the moved lanes onto a corner or an edge of the other lanes. Without one
   * in reach, it aligns the edges and the center of the moved lanes as a whole, and an axis that aligns with nothing
   * keeps even gaps to the neighbors. A drag kept on one axis only snaps along it.
   */
  function snapLanes(
    moving: Array<[PixelLane, PixelLane]>,
    initialBounds: Bounds,
    targets: SnapTargets,
    offset: Point,
    constrained: boolean,
  ): Point {
    const corners = moving.flatMap(([, initial]) =>
      CORNERS.map((corner) => ({
        x: initial[corner].x + offset.x,
        y: initial[corner].y + offset.y,
      })),
    );
    const snapped = snapOffset(corners, targets.points, snapDistance());
    if (snapped.target && !constrained) {
      feedback.value = {
        point: snapped.target,
        lines: [],
        gaps: [],
      };
      return {
        x: offset.x + snapped.offset.x,
        y: offset.y + snapped.offset.y,
      };
    }

    const moved = translateBounds(initialBounds, offset);
    const aligned = snapBounds(moved, targets.axes, snapDistance());
    const spacing = spacingTargets(moved, targets.others);
    const evenX = snapValue([moved.minX], spacing.x, snapDistance());
    const evenY = snapValue([moved.minY], spacing.y, snapDistance());
    const alignedX = aligned.feedback.lines.some((line) => line.axis === 'x');
    const alignedY = aligned.feedback.lines.some((line) => line.axis === 'y');
    feedback.value = aligned.feedback;

    // A drag kept on one axis does not leave it for a snap
    const keepX = constrained && offset.x === 0;
    const keepY = constrained && offset.y === 0;
    return {
      x: keepX ? 0 : offset.x + (alignedX ? aligned.offset.x : evenX.delta),
      y: keepY ? 0 : offset.y + (alignedY ? aligned.offset.y : evenY.delta),
    };
  }

  /**
   * startLaneDrag selects a lane and starts moving the selection. Shift, Ctrl, or Cmd toggle the lane
   * in the selection. Pressing on an already selected lane keeps the selection, so a group moves together.
   */
  function startLaneDrag(event: PointerEvent, laneKey: string): void {
    event.preventDefault();

    presetForm.editor.selection.press(laneKey, isAdditiveSelect(event));
    const selection = presetForm.editor.selection.selected.value;
    if (selection.size === 0 || !selection.has(laneKey)) return;

    state = {
      mode: 'lane',
      initialLanes: structuredClone(toRaw(presetForm.preset.lanes)),
      start: options.toFrame(event.clientX, event.clientY),
      // The moved lanes cannot snap onto themselves
      targets: targetsExcluding(selection),
    };
    start('lane', event);
  }

  /** startCornerDrag begins dragging a corner of a lane from its current position. */
  function startCornerDrag(event: PointerEvent, laneKey: string, corner: CornerKey): void {
    event.preventDefault();
    const lane = presetForm.preset.lanes[laneKey];
    if (!lane) return;

    state = {
      mode: 'corner',
      laneKey,
      corner,
      start: {
        x: lane[corner].x,
        y: lane[corner].y,
      },
      targets: targetsExcluding(new Set([laneKey])),
    };
    start('corner', event);
  }

  /** start begins a gesture, during which the shortcuts and the automatic history checkpoint wait. */
  function start(next: DragMode, event: PointerEvent): void {
    mode.value = next;
    presetForm.editor.dragging.value = true;
    pointerDrag.start(event);
  }

  return {
    mode,
    feedback,
    startLaneDrag,
    startCornerDrag,
  };
}
