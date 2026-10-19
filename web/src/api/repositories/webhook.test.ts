import { describe, expect, it } from 'vitest';

import { webhookRepository } from '@/api/repositories/webhook';
import type { WebhookCreateRequest, WebhookUpdateRequest } from '@/api/types';
import { jsonResponse, useFetchMock } from '@/test/fetchMock';

describe('webhookRepository', () => {
  const { fetchMock, lastRequest } = useFetchMock();

  describe('list', () => {
    it('returns the webhooks', async () => {
      const webhooks = [
        {
          id: 'a',
        },
      ];
      fetchMock.mockResolvedValue(jsonResponse(200, webhooks));

      const result = await webhookRepository.list();

      expect(result).toEqual(webhooks);
      expect(lastRequest().url).toBe('/api/webhooks');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('get', () => {
    it('returns the webhook under its encoded id', async () => {
      fetchMock.mockResolvedValue(
        jsonResponse(200, {
          id: 'id a/b',
        }),
      );

      const result = await webhookRepository.get('id a/b');

      expect(result).toEqual({
        id: 'id a/b',
      });
      expect(lastRequest().url).toBe('/api/webhooks/id%20a%2Fb');
      expect(lastRequest().init?.method).toBe('GET');
    });
  });

  describe('create', () => {
    it('stores the webhook', async () => {
      const payload = {
        name: 'website',
      } as WebhookCreateRequest;
      fetchMock.mockResolvedValue(jsonResponse(201, payload));

      const result = await webhookRepository.create(payload);

      expect(result).toEqual(payload);
      expect(lastRequest().url).toBe('/api/webhooks');
      expect(lastRequest().init?.method).toBe('POST');
      expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
    });
  });

  describe('update', () => {
    it('replaces the webhook under its encoded id', async () => {
      const payload = {
        name: 'website',
      } as WebhookUpdateRequest;
      fetchMock.mockResolvedValue(jsonResponse(200, payload));

      const result = await webhookRepository.update('id a/b', payload);

      expect(result).toEqual(payload);
      expect(lastRequest().url).toBe('/api/webhooks/id%20a%2Fb');
      expect(lastRequest().init?.method).toBe('PUT');
      expect(lastRequest().init?.body).toBe(JSON.stringify(payload));
    });
  });

  describe('delete', () => {
    it('removes the webhook under its encoded id', async () => {
      fetchMock.mockResolvedValue(jsonResponse(204, undefined));

      await webhookRepository.delete('id a/b');

      expect(lastRequest().url).toBe('/api/webhooks/id%20a%2Fb');
      expect(lastRequest().init?.method).toBe('DELETE');
    });
  });

  describe('test', () => {
    it('sends a test message to the webhook under its encoded id', async () => {
      const delivery = {
        outcome: 'delivered',
      };
      fetchMock.mockResolvedValue(jsonResponse(200, delivery));

      const result = await webhookRepository.test('id a/b');

      expect(result).toEqual(delivery);
      expect(lastRequest().url).toBe('/api/webhooks/id%20a%2Fb/test');
      expect(lastRequest().init?.method).toBe('POST');
    });
  });
});
