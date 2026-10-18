import { defineStore } from 'pinia';
import { ref } from 'vue';

import { metaRepository } from '@/api/repositories/meta';
import type { MetaResponse } from '@/api/types';

/**
 * useMetaStore caches the metadata of the application from /api/meta, such as its name, version,
 * language, and theme, which every page needs before it renders.
 */
export const useMetaStore = defineStore('meta', () => {
  const meta = ref<MetaResponse | null>(null);

  let inflight: Promise<MetaResponse> | null = null;
  // Requests are numbered, so a response only replaces the cache if no newer response did already
  let requested = 0;
  let applied = 0;

  /** load fetches /api/meta once and caches it for the lifetime of this page load, or until reload. */
  async function load(): Promise<MetaResponse> {
    if (meta.value) return meta.value;
    if (inflight) return inflight;
    return fetchMeta();
  }

  /**
   * reload fetches /api/meta again and replaces the cache, for example after the server restarted with another
   * configuration. The cached metadata stays until the new one arrives.
   */
  function reload(): Promise<MetaResponse> {
    return fetchMeta();
  }

  /**
   * fetchMeta requests the metadata and caches it. A response replaces the cache unless the response of a newer
   * request already did, so a newer request that fails keeps the result of an older one.
   */
  function fetchMeta(): Promise<MetaResponse> {
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
