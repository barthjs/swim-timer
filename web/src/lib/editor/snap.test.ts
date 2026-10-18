import { describe, expect, it } from 'vitest';

import {
  axisFeedback,
  boundsTargets,
  mergeTargets,
  closestPointOnSegment,
  noSnap,
  snapBounds,
  snapOffset,
  snapPoint,
  snapPointToAxes,
  snapValue,
  type PointTargets,
  type Segment,
} from '@/lib/editor/snap';

// The corners and edges of a rectangle from 100, 100 to 900, 150 and the border of a 1920 x 1080 content.
const pointTargets: PointTargets = {
  corners: [
    {
      x: 100,
      y: 100,
    },
    {
      x: 900,
      y: 100,
    },
    {
      x: 900,
      y: 150,
    },
    {
      x: 100,
      y: 150,
    },
  ],
  edges: [
    [
      {
        x: 100,
        y: 150,
      },
      {
        x: 900,
        y: 150,
      },
    ],
    [
      {
        x: 1920,
        y: 0,
      },
      {
        x: 1920,
        y: 1080,
      },
    ],
  ],
};

const axisTargets = {
  x: [0, 50, 100],
  y: [0, 30, 60],
};

describe('noSnap', () => {
  it('reports neither a point nor a line', () => {
    const feedback = noSnap();

    expect(feedback).toEqual({
      point: null,
      lines: [],
      gaps: [],
    });
  });
});

describe('boundsTargets', () => {
  it('collects the edges and centers of the bounds and of the content', () => {
    const bounds = [
      {
        minX: 10,
        maxX: 30,
        minY: 40,
        maxY: 80,
      },
    ];

    const targets = boundsTargets(bounds, {
      width: 200,
      height: 100,
    });

    expect(targets).toEqual({
      x: [0, 100, 200, 10, 20, 30],
      y: [0, 50, 100, 40, 60, 80],
    });
  });

  it('lists a shared line once', () => {
    const bounds = [
      {
        minX: 0,
        maxX: 200,
        minY: 0,
        maxY: 100,
      },
    ];

    const targets = boundsTargets(bounds, {
      width: 200,
      height: 100,
    });

    expect(targets).toEqual({
      x: [0, 100, 200],
      y: [0, 50, 100],
    });
  });
});

describe('mergeTargets', () => {
  it('combines the lines of every source', () => {
    const merged = mergeTargets(
      {
        x: [10],
        y: [20],
      },
      {
        x: [30],
        y: [],
      },
    );

    expect(merged).toEqual({
      x: [10, 30],
      y: [20],
    });
  });
});

describe('closestPointOnSegment', () => {
  const segment: Segment = [
    {
      x: 0,
      y: 0,
    },
    {
      x: 100,
      y: 0,
    },
  ];

  it('projects the point onto the segment', () => {
    const point = {
      x: 40,
      y: 7,
    };

    const closest = closestPointOnSegment(point, segment);

    expect(closest).toEqual({
      x: 40,
      y: 0,
    });
  });

  it('clamps the point to the end of the segment', () => {
    const point = {
      x: 140,
      y: 7,
    };

    const closest = closestPointOnSegment(point, segment);

    expect(closest).toEqual({
      x: 100,
      y: 0,
    });
  });
});

describe('snapPoint', () => {
  it('snaps onto a corner in reach', () => {
    const point = {
      x: 905,
      y: 154,
    };

    const snapped = snapPoint(point, pointTargets, 8);

    expect(snapped).toMatchObject({
      point: {
        x: 900,
        y: 150,
      },
      target: {
        x: 900,
        y: 150,
      },
    });
  });

  it('prefers a corner over a closer edge', () => {
    const point = {
      x: 895,
      y: 151,
    };

    const snapped = snapPoint(point, pointTargets, 8);

    expect(snapped.point).toEqual({
      x: 900,
      y: 150,
    });
  });

  it('snaps onto an edge in reach', () => {
    const point = {
      x: 500,
      y: 155,
    };

    const snapped = snapPoint(point, pointTargets, 8);

    expect(snapped.point).toEqual({
      x: 500,
      y: 150,
    });
  });

  it('keeps a point without a target in reach', () => {
    const point = {
      x: 500,
      y: 400,
    };

    const snapped = snapPoint(point, pointTargets, 8);

    expect(snapped).toMatchObject({
      point: {
        x: 500,
        y: 400,
      },
      target: null,
    });
  });
});

describe('snapOffset', () => {
  it('shifts the points by the offset of the closest snap', () => {
    const corners = [
      {
        x: 100,
        y: 156,
      },
      {
        x: 903,
        y: 153,
      },
    ];

    const snapped = snapOffset(corners, pointTargets, 8);

    expect(snapped).toEqual({
      offset: {
        x: -3,
        y: -3,
      },
      target: {
        x: 900,
        y: 150,
      },
    });
  });

  it('returns no shift without a target in reach', () => {
    const corners = [
      {
        x: 500,
        y: 400,
      },
    ];

    const snapped = snapOffset(corners, pointTargets, 8);

    expect(snapped).toEqual({
      offset: {
        x: 0,
        y: 0,
      },
      target: null,
    });
  });
});

describe('snapValue', () => {
  it('shifts the closest value onto its target line', () => {
    const snapped = snapValue([47, 62], [50, 60], 5);

    expect(snapped).toEqual({
      delta: -2,
      line: 60,
    });
  });

  it('returns no shift without a line in reach', () => {
    const snapped = snapValue([20], [50, 60], 5);

    expect(snapped).toEqual({
      delta: 0,
      line: null,
    });
  });
});

describe('snapBounds', () => {
  it('aligns the center and an edge and marks where the lines cross', () => {
    const bounds = {
      minX: 38,
      maxX: 58,
      minY: 2,
      maxY: 12,
    };

    const snapped = snapBounds(bounds, axisTargets, 3);

    expect(snapped).toEqual({
      offset: {
        x: 2,
        y: -2,
      },
      feedback: {
        point: {
          x: 50,
          y: 0,
        },
        lines: [
          {
            axis: 'x',
            value: 50,
          },
          {
            axis: 'y',
            value: 0,
          },
        ],
        gaps: [],
      },
    });
  });

  it('aligns one axis without a point', () => {
    const bounds = {
      minX: 98,
      maxX: 99,
      minY: 10,
      maxY: 20,
    };

    const snapped = snapBounds(bounds, axisTargets, 3);

    expect(snapped.feedback).toEqual({
      point: null,
      lines: [
        {
          axis: 'x',
          value: 100,
        },
      ],
      gaps: [],
    });
  });

  it('keeps bounds without a line in reach', () => {
    const bounds = {
      minX: 10,
      maxX: 20,
      minY: 10,
      maxY: 20,
    };

    const snapped = snapBounds(bounds, axisTargets, 3);

    expect(snapped).toEqual({
      offset: {
        x: 0,
        y: 0,
      },
      feedback: noSnap(),
    });
  });
});

describe('snapPointToAxes', () => {
  it('moves the point onto the lines in reach', () => {
    const point = {
      x: 52,
      y: 10,
    };

    const snapped = snapPointToAxes(point, axisTargets, 3);

    expect(snapped).toEqual({
      point: {
        x: 50,
        y: 10,
      },
      feedback: {
        point: null,
        lines: [
          {
            axis: 'x',
            value: 50,
          },
        ],
        gaps: [],
      },
    });
  });
});

describe('axisFeedback', () => {
  it('marks the crossing of two lines', () => {
    const feedback = axisFeedback(10, 20);

    expect(feedback.point).toEqual({
      x: 10,
      y: 20,
    });
  });

  it('shows a single line without a point', () => {
    const feedback = axisFeedback(null, 20);

    expect(feedback).toEqual({
      point: null,
      lines: [
        {
          axis: 'y',
          value: 20,
        },
      ],
      gaps: [],
    });
  });
});
