import { useI18n } from 'vue-i18n';

import type { WebhookDeliveryErrorCode } from '@/api/types';

/** EVENT_LABEL_KEYS are the translation keys of the event names, including the message of a test delivery. */
const EVENT_LABEL_KEYS: Record<string, string> = {
  'heat.updated': 'pages.config.webhook.events.heatUpdated',
  'meet.reloaded': 'pages.config.webhook.events.meetReloaded',
  'webhook.test': 'pages.config.webhook.events.test',
};

/** useWebhookTexts translates the event names and the reasons of failed deliveries the server reports as codes. */
export function useWebhookTexts() {
  const { t } = useI18n();

  /** eventLabel returns the name of an event, or the event itself for one the interface does not know. */
  function eventLabel(event: string): string {
    const key = EVENT_LABEL_KEYS[event];
    return key ? t(key) : event;
  }

  /** problem describes a failed delivery by the reason it got no response, or by the status of the response. */
  function problem(
    errorCode: WebhookDeliveryErrorCode | '' | undefined,
    statusCode: number | undefined,
  ): string {
    if (errorCode) return t(`pages.config.webhook.errors.${errorCode}`);
    return t('pages.config.webhook.errors.status', {
      status: statusCode ?? 0,
    });
  }

  return {
    eventLabel,
    problem,
  };
}
