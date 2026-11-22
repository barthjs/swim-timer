import { toValue, type MaybeRefOrGetter } from 'vue';
import { useI18n } from 'vue-i18n';

import type { WebhookDeliveryErrorCode } from '@/api/types/webhookDelivery';
import type { WebhookEventOption } from '@/lib/webhook/webhook';

const TEST_EVENT = 'webhook.test';

/** Translates the event names and the reasons of failed deliveries the server reports as codes. */
export function useWebhookTexts(events: MaybeRefOrGetter<WebhookEventOption[]>) {
  const { t, te } = useI18n();

  /** Returns the name of an event, or the event itself for one the interface does not know. */
  function eventLabel(event: string): string {
    if (event === TEST_EVENT) return t('webhook.test');
    return toValue(events).find((option) => option.value === event)?.label ?? event;
  }

  /**
   * Describes a failed delivery by the reason it got no response, or by the status of the response.
   * A reason the interface does not know yet falls back to the status.
   */
  function problem(
    errorCode: WebhookDeliveryErrorCode | '' | undefined,
    statusCode: number | undefined,
  ): string {
    const key = `webhook.errors.${errorCode}`;
    if (errorCode && te(key)) return t(key);
    return t('webhook.errors.status', {
      status: statusCode ?? 0,
    });
  }

  return {
    eventLabel,
    problem,
  };
}
