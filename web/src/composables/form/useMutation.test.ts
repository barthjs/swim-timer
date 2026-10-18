import type { FormError } from '@nuxt/ui';
import type { useToast } from '@nuxt/ui/composables';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiError } from '@/api/errors';
import type { FormErrorTarget } from '@/composables/form/useFormValidation';
import { useMutation } from '@/composables/form/useMutation';

const toast = {
  add: vi.fn<ReturnType<typeof useToast>['add']>(),
};

vi.mock('@nuxt/ui/composables', () => ({
  useToast: () => toast,
}));

vi.mock('@/composables/useApiError', () => ({
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

/** createForm returns a form double that records the errors it receives. */
function createForm() {
  const form = {
    setErrors: vi.fn<FormErrorTarget['setErrors']>(),
    clear: vi.fn<() => void>(),
  };
  return form satisfies FormErrorTarget;
}

describe('useMutation', () => {
  beforeEach(() => {
    toast.add.mockClear();
    mapFieldErrors.mockClear();
  });

  describe('mutate', () => {
    it('returns the result, shows the success toast, and runs onSuccess', async () => {
      const onSuccess = vi.fn<(result: string) => void>();
      const { mutate } = useMutation((name: string) => Promise.resolve(`saved ${name}`), {
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
      const { mutate, pending } = useMutation(() => Promise.resolve('saved'));

      const promise = mutate();

      expect(pending.value).toBe(true);
      await promise;
      expect(pending.value).toBe(false);
    });

    it('shows no success toast without a success message', async () => {
      const { mutate } = useMutation(() => Promise.resolve(1));

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
      const { mutate, error: lastError } = useMutation(() => Promise.reject(error), {
        form,
      });

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
      const { mutate } = useMutation(() => Promise.reject(error), {
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
      const { mutate } = useMutation(() => Promise.reject(error));

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
      const { mutate, errorMessage } = useMutation(() => Promise.reject(error));

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
      const { mutate } = useMutation(() => Promise.reject(new Error('x')), {
        onError: () => true,
      });

      await mutate();

      expect(toast.add).not.toHaveBeenCalled();
    });

    it('shows no error toast when disabled', async () => {
      const { mutate, errorMessage } = useMutation(() => Promise.reject(new Error('x')), {
        toast: false,
      });

      await mutate();

      expect(toast.add).not.toHaveBeenCalled();
      expect(errorMessage.value).toBe('error:unknown');
    });

    it('clears the previous error when retrying', async () => {
      let fail = true;
      const { mutate, error } = useMutation(() =>
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
