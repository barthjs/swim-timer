import { faker } from '@faker-js/faker';
import type { FormError } from '@nuxt/ui';
import type { useToast } from '@nuxt/ui/composables';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { nextTick, reactive } from 'vue';

import { ApiError } from '@/api/client/errors';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { useMutation } from '@/composables/form/useMutation';

const toast = {
  add: vi.fn<ReturnType<typeof useToast>['add']>(),
  promise: vi.fn<ReturnType<typeof useToast>['promise']>(async (promise) => promise),
};

vi.mock('@nuxt/ui/composables', () => ({
  useToast: () => toast,
}));

vi.mock('@/composables/form/useApiError', () => ({
  useApiError: () => ({
    messageFor: (err: unknown) => (err instanceof ApiError ? `error:${err.code}` : 'error:unknown'),
  }),
}));

const mapFieldErrors = vi.fn<(fields: Record<string, unknown>, values?: unknown) => FormError[]>(
  (fields) =>
    Object.keys(fields).map((name) => ({
      name,
      message: `${name} invalid`,
    })),
);

vi.mock('@/composables/form/useValidationMessages', () => ({
  useValidationMessages: () => ({
    mapFieldErrors,
  }),
}));

/** Returns a form double that records the errors it receives. */
function createForm() {
  const form = {
    setErrors: vi.fn<FormErrorTarget['setErrors']>(),
    clear: vi.fn<(name?: string) => void>(),
  };
  return form satisfies FormErrorTarget;
}

describe('useMutation', () => {
  beforeEach(() => {
    toast.add.mockClear();
    toast.promise.mockClear();
    mapFieldErrors.mockClear();
  });

  it('clears the error of an edited field', async () => {
    const form = createForm();
    const values = reactive({
      name: faker.lorem.word(),
      email: faker.internet.email(),
    });
    useMutation(async () => Promise.resolve(), {
      form,
      values,
    });

    values.name = '';
    await nextTick();

    expect(form.clear).toHaveBeenCalledExactlyOnceWith('name');
  });

  describe('isPending', () => {
    it('returns whether a call with the key runs', async () => {
      const key = faker.string.uuid();
      const { mutate, isPending } = useMutation(async (id: string) => Promise.resolve(id), {
        key: (id) => id,
      });

      const promise = mutate(key);

      expect(isPending(key)).toBe(true);
      expect(isPending(faker.string.uuid())).toBe(false);
      await promise;
      expect(isPending(key)).toBe(false);
    });
  });

  describe('mutate', () => {
    it('returns the result, shows the success toast, and runs onSuccess', async () => {
      const onSuccess = vi.fn<(result: string) => void>();
      const { mutate } = useMutation(async (name: string) => Promise.resolve(`saved ${name}`), {
        success: (result) => result,
        onSuccess,
      });

      const result = await mutate('item');

      expect(result).toBe('saved item');
      expect(onSuccess).toHaveBeenCalledWith('saved item', 'item');
      expect(toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'saved item',
          color: 'success',
        }),
      );
    });

    it('is pending while the mutation runs', async () => {
      const { mutate, pending } = useMutation(async () => Promise.resolve('saved'));

      const promise = mutate();

      expect(pending.value).toBe(true);
      await promise;
      expect(pending.value).toBe(false);
    });

    it('ignores a call while the request is pending', async () => {
      const request = vi.fn<() => Promise<string>>(async () => Promise.resolve('saved'));
      const { mutate } = useMutation(request);
      const first = mutate();

      const second = await mutate();

      await first;
      expect(second).toBeUndefined();
      expect(request).toHaveBeenCalledOnce();
    });

    it('ignores a call with the same key while it is pending', async () => {
      const key = faker.string.uuid();
      const request = vi.fn<(id: string) => Promise<string>>(async (id) => Promise.resolve(id));
      const { mutate } = useMutation(request, {
        key: (id) => id,
      });
      const first = mutate(key);

      const second = await mutate(key);

      await first;
      expect(second).toBeUndefined();
      expect(request).toHaveBeenCalledOnce();
    });

    it('runs a call with another key while one is pending', async () => {
      const [firstKey, secondKey] = [faker.string.uuid(), faker.string.uuid()];
      const request = vi.fn<(id: string) => Promise<string>>(async (id) => Promise.resolve(id));
      const { mutate } = useMutation(request, {
        key: (id) => id,
      });
      const first = mutate(firstKey);

      const second = await mutate(secondKey);

      await first;
      expect(second).toBe(secondKey);
      expect(request).toHaveBeenCalledTimes(2);
    });

    it('is pending while any keyed call runs', async () => {
      const [firstKey, secondKey] = [faker.string.uuid(), faker.string.uuid()];
      let resolveFirst = (): void => {};
      const { mutate, pending } = useMutation(
        async (id: string) =>
          id === firstKey
            ? new Promise<string>((resolve) => {
                resolveFirst = () => resolve(id);
              })
            : Promise.resolve(id),
        {
          key: (id) => id,
        },
      );
      const first = mutate(firstKey);

      await mutate(secondKey);

      expect(pending.value).toBe(true);
      resolveFirst();
      await first;
      expect(pending.value).toBe(false);
    });

    it('runs the request once the confirmation is given', async () => {
      const confirm = vi.fn<(name: string) => Promise<boolean>>(async () => Promise.resolve(true));
      const { mutate } = useMutation(async (name: string) => Promise.resolve(`saved ${name}`), {
        confirm,
      });

      const result = await mutate('item');

      expect(result).toBe('saved item');
      expect(confirm).toHaveBeenCalledWith('item');
    });

    it('skips the request without the confirmation', async () => {
      const request = vi.fn<() => Promise<string>>(async () => Promise.resolve('saved'));
      const { mutate, pending } = useMutation(request, {
        confirm: async () => Promise.resolve(false),
      });

      const result = await mutate();

      expect(result).toBeUndefined();
      expect(request).not.toHaveBeenCalled();
      expect(pending.value).toBe(false);
    });

    it('shows a loading toast that turns into the success toast', async () => {
      const loading = faker.lorem.words();
      const success = faker.lorem.words();
      const result = faker.lorem.word();
      const { mutate } = useMutation(async () => result, {
        loading,
        success,
      });

      await mutate();

      const options = toast.promise.mock.calls[0]?.[1];
      expect(options?.loading).toEqual({
        title: loading,
      });
      expect(typeof options?.success === 'function' ? options.success(result) : undefined).toEqual({
        title: success,
        icon: 'i-lucide-check-circle',
        color: 'success',
      });
      expect(toast.add).not.toHaveBeenCalled();
    });

    it('shows the error toast after the loading toast of a failed request', async () => {
      const { mutate } = useMutation(
        async () =>
          Promise.reject(
            new ApiError({
              message: faker.lorem.words(),
              status: 500,
              code: 'internal',
            }),
          ),
        {
          loading: faker.lorem.words(),
        },
      );

      await mutate();

      expect(toast.promise.mock.calls[0]?.[1].error).toBeUndefined();
      expect(toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'error:internal',
          color: 'error',
        }),
      );
    });

    it('shows no success toast without a success message', async () => {
      const { mutate } = useMutation(async () => Promise.resolve(1));

      await mutate();

      expect(toast.add).not.toHaveBeenCalled();
    });

    it('routes field errors to the form instead of a toast', async () => {
      const form = createForm();
      const error = new ApiError({
        message: 'invalid',
        status: 422,
        code: 'validation_failed',
        fields: {
          name: [
            {
              tag: 'required',
            },
          ],
        },
      });
      const { mutate, error: lastError } = useMutation(
        async (): Promise<unknown> => Promise.reject(error),
        {
          form,
        },
      );

      const result = await mutate();

      expect(result).toBeUndefined();
      expect(form.clear).toHaveBeenCalledOnce();
      expect(form.setErrors).toHaveBeenCalledWith([
        {
          name: 'name',
          message: 'name invalid',
        },
      ]);
      expect(toast.add).not.toHaveBeenCalled();
      expect(lastError.value).toBe(error);
    });

    it('maps field errors against the given values', async () => {
      const values = {
        name: 'x',
      };
      const fields = {
        name: [
          {
            tag: 'min',
            param: '3',
          },
        ],
      };
      const error = new ApiError({
        message: 'invalid',
        status: 422,
        code: 'validation_failed',
        fields,
      });
      const { mutate } = useMutation(async (): Promise<unknown> => Promise.reject(error), {
        form: createForm(),
        values,
      });

      await mutate();

      expect(mapFieldErrors).toHaveBeenCalledWith(fields, values);
    });

    it('names the first failed field in the toast of a validation error without a form', async () => {
      const error = new ApiError({
        message: 'invalid',
        status: 422,
        code: 'validation_failed',
        fields: {
          file: [
            {
              tag: 'too_large',
            },
          ],
          name: [
            {
              tag: 'required',
            },
          ],
        },
      });
      const { mutate } = useMutation(async () => Promise.reject(error));

      await mutate();

      expect(toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'error:validation_failed',
          description: 'file invalid',
          color: 'error',
        }),
      );
    });

    it('shows other errors as a toast and exposes the message', async () => {
      const error = new ApiError({
        message: 'down',
        status: 500,
        code: 'server_error',
      });
      const { mutate, errorMessage } = useMutation(async () => Promise.reject(error));

      await mutate();

      expect(errorMessage.value).toBe('error:server_error');
      expect(toast.add).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'error:server_error',
          color: 'error',
        }),
      );
    });

    it('lets onError suppress the toast', async () => {
      const { mutate } = useMutation(async () => Promise.reject(new Error('x')), {
        onError: () => true,
      });

      await mutate();

      expect(toast.add).not.toHaveBeenCalled();
    });

    it('shows no error toast when disabled', async () => {
      const { mutate, errorMessage } = useMutation(async () => Promise.reject(new Error('x')), {
        toast: false,
      });

      await mutate();

      expect(toast.add).not.toHaveBeenCalled();
      expect(errorMessage.value).toBe('error:unknown');
    });

    it('clears the previous error when retrying', async () => {
      let fail = true;
      const { mutate, error } = useMutation(async () =>
        fail ? Promise.reject(new Error('x')) : Promise.resolve('ok'),
      );
      await mutate();
      expect(error.value).not.toBeNull();
      fail = false;

      await mutate();

      expect(error.value).toBeNull();
    });
  });
});
