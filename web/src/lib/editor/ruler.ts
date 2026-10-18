/** RULER_THICKNESS is the thickness of a ruler on the screen, in pixels. */
export const RULER_THICKNESS = 20;

/** The smallest distance on the screen between two labeled ticks and between two ticks at all. */
const LABEL_SPACING_MIN = 60;
const TICK_SPACING_MIN = 5;

/** The lengths of the ticks on the screen, from labeled to unlabeled. */
const MAJOR_TICK = 12;
const MEDIUM_TICK = 8;
const MINOR_TICK = 4;

/** The divisions of a labeled step into minor steps, from the finest one. */
const SUBDIVISIONS = [10, 5, 2, 1];

export type RulerTick = {
  // The position on the screen along the ruler, in pixels from its start.
  position: number;
  // The length of the tick on the screen.
  size: number;
  // The value written next to a labeled tick, or null.
  label: string | null;
};

/**
 * rulerTicks returns the ticks of a ruler of the given length on the screen. A value v of the content lies
 * at offset + v * scale on the screen. The steps between the labeled ticks are 1, 2, or 5 times a power of ten,
 * chosen so the labels keep a readable distance at every zoom.
 */
export function rulerTicks(length: number, offset: number, scale: number): RulerTick[] {
  if (length <= 0 || scale <= 0) return [];

  const major = majorStep(scale);
  const subdivision =
    SUBDIVISIONS.find((count) => (major / count) * scale >= TICK_SPACING_MIN) ?? 1;
  const minor = major / subdivision;

  const first = Math.ceil(-offset / scale / minor);
  const last = Math.floor((length - offset) / scale / minor);
  const ticks: RulerTick[] = [];

  for (let index = first; index <= last; index++) {
    const value = index * minor;
    const position = offset + value * scale;
    if (index % subdivision === 0) {
      ticks.push({
        position,
        size: MAJOR_TICK,
        label: String(Math.round(value)),
      });
    } else if (subdivision % 2 === 0 && index % (subdivision / 2) === 0) {
      ticks.push({
        position,
        size: MEDIUM_TICK,
        label: null,
      });
    } else {
      ticks.push({
        position,
        size: MINOR_TICK,
        label: null,
      });
    }
  }
  return ticks;
}

/** majorStep returns the smallest step of 1, 2, or 5 times a power of ten that keeps labels apart. */
function majorStep(scale: number): number {
  const minimum = LABEL_SPACING_MIN / scale;
  let power = 10 ** Math.floor(Math.log10(minimum));
  for (;;) {
    for (const factor of [1, 2, 5]) {
      if (factor * power >= minimum) return factor * power;
    }
    power *= 10;
  }
}
