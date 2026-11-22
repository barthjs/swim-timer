/**
 * Compares the running version with the latest release.
 * unsupported names why this installation cannot install it itself, missing if it can.
 */
export type UpdateCheckResponse = {
  current: string;
  latest: string;
  url: string;
  newer: boolean;
  unsupported?: 'development_build' | 'container' | 'package_managed' | 'not_writable';
};

export type UpdateStatusResponse = {
  state: 'idle' | 'running' | 'failed' | 'applied';
  version?: string;
  error?: string;
};
