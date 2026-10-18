import { useI18n } from 'vue-i18n';

import { ApiError } from '@/api/errors';

/**
 * useApiError resolves a localized message for an API error.
 * It falls back to the generic server error message when the value is not an ApiError.
 */
export function useApiError() {
  const { t, te } = useI18n();

  /** messageFor returns the localized message for an error. */
  const messageFor = (err: unknown): string => {
    if (err instanceof ApiError && err.code === 'rate_limited' && err.retryAfter !== undefined) {
      return retryMessage(err.retryAfter);
    }
    if (err instanceof ApiError) {
      const key = `errors.${err.code}`;
      if (te(key)) return t(key);
    }
    return t('errors.server_error');
  };

  /**
   * retryMessage names the wait in seconds below a minute and in whole minutes above, rounded up,
   * so the user never retries too early.
   */
  function retryMessage(seconds: number): string {
    if (seconds < 60) {
      return t(
        'errors.rate_limited_seconds',
        {
          count: seconds,
        },
        seconds,
      );
    }
    const minutes = Math.ceil(seconds / 60);
    return t(
      'errors.rate_limited_minutes',
      {
        count: minutes,
      },
      minutes,
    );
  }

  return {
    messageFor,
  };
}
