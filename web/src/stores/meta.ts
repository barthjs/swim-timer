import { defineStore } from 'pinia';
import { ref } from 'vue';

import { metaRepository } from '@/api/repositories/meta';
import type { MetaResponse } from '@/api/types/meta';

/** Caches the metadata of the application from /api/meta, which every page needs before it renders. */
export const useMetaStore = defineStore('meta', () => {
  const meta = ref<MetaResponse | null>(null);

  let inflight: Promise<MetaResponse> | null = null;
  // Requests are numbered, so a response only replaces the cache if no newer response did already
  let requested = 0;
  let applied = 0;

  /** Fetches /api/meta once and caches it for the lifetime of this page load, or until reload. */
  async function load(): Promise<MetaResponse> {
    if (meta.value) return meta.value;
    if (inflight) return inflight;
    return fetchMeta();
  }

  /** Fetches /api/meta again and replaces the cache once the new metadata arrives. */
  async function reload(): Promise<MetaResponse> {
    return fetchMeta();
  }

  /**
   * Requests the metadata and caches it. A response replaces the cache unless the response of a newer request already did,
   * so a newer request that fails keeps the result of an older one.
   */
  async function fetchMeta(): Promise<MetaResponse> {
    requested += 1;
    const sequence = requested;
    const request = metaRepository
      .get()
      .then((result) => {
        if (sequence > applied) {
          meta.value = result;
          applied = sequence;
        }
        return result;
      })
      .finally(() => {
        if (inflight === request) inflight = null;
      });
    inflight = request;
    return request;
  }

  return {
    meta,
    load,
    reload,
  };
});
