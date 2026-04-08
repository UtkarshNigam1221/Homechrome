import { useMutation, useQueryClient } from '@tanstack/react-query';
import toast from 'react-hot-toast';

import { getErrorMessage } from '@/shared/api/client';

export interface UseFormMutationOptions<TCreate, TUpdate, TResponse> {
  queryKey: string;
  createFn: (data: TCreate) => Promise<TResponse>;
  updateFn: (id: string, data: TUpdate) => Promise<TResponse>;
  entityName: string;
  onSuccess: () => void;
}

export interface UseFormMutationResult<TCreate, TUpdate> {
  createMutation: ReturnType<typeof useMutation<unknown, Error, TCreate>>;
  updateMutation: ReturnType<typeof useMutation<unknown, Error, { id: string; data: TUpdate }>>;
  isLoading: boolean;
  onSubmit: (id: string | undefined, createData: TCreate, updateData: TUpdate) => void;
}

export function useFormMutation<TCreate, TUpdate, TResponse = unknown>({
  queryKey,
  createFn,
  updateFn,
  entityName,
  onSuccess,
}: UseFormMutationOptions<TCreate, TUpdate, TResponse>): UseFormMutationResult<TCreate, TUpdate> {
  const queryClient = useQueryClient();

  const createMutation = useMutation<TResponse, Error, TCreate>({
    mutationFn: createFn,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [queryKey] });
      toast.success(`${entityName} created successfully`);
      onSuccess();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const updateMutation = useMutation<TResponse, Error, { id: string; data: TUpdate }>({
    mutationFn: ({ id, data }) => updateFn(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [queryKey] });
      toast.success(`${entityName} updated successfully`);
      onSuccess();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const isLoading = createMutation.isPending || updateMutation.isPending;

  const onSubmit = (id: string | undefined, createData: TCreate, updateData: TUpdate) => {
    if (id) {
      updateMutation.mutate({ id, data: updateData });
    } else {
      createMutation.mutate(createData);
    }
  };

  return {
    createMutation,
    updateMutation,
    isLoading,
    onSubmit,
  };
}
