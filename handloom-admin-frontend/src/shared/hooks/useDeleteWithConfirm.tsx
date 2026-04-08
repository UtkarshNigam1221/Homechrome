import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useCallback, useState } from 'react';
import toast from 'react-hot-toast';

import { getErrorMessage } from '@/shared/api/client';
import { ConfirmModal } from '@/shared/components/ui';

export interface UseDeleteWithConfirmOptions<T> {
  queryKey: string;
  deleteFn: (id: string) => Promise<void>;
  entityName: string;
  getEntityName: (entity: T) => string;
}

export interface UseDeleteWithConfirmResult<T> {
  deleteTarget: T | null;
  setDeleteTarget: (entity: T | null) => void;
  confirmDelete: () => void;
  isDeleting: boolean;
  DeleteConfirmModal: () => React.ReactElement;
}

export function useDeleteWithConfirm<T extends { id: string }>({
  queryKey,
  deleteFn,
  entityName,
  getEntityName,
}: UseDeleteWithConfirmOptions<T>): UseDeleteWithConfirmResult<T> {
  const queryClient = useQueryClient();
  const [deleteTarget, setDeleteTarget] = useState<T | null>(null);

  const deleteMutation = useMutation<void, Error, string>({
    mutationFn: deleteFn,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [queryKey] });
      toast.success(`${entityName} deleted successfully`);
      setDeleteTarget(null);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const confirmDelete = useCallback(() => {
    if (deleteTarget) {
      deleteMutation.mutate(deleteTarget.id);
    }
  }, [deleteTarget, deleteMutation]);

  const DeleteConfirmModal = useCallback(
    () => (
      <ConfirmModal
        isOpen={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={confirmDelete}
        title={`Delete ${entityName}`}
        message={
          deleteTarget
            ? `Are you sure you want to delete "${getEntityName(deleteTarget)}"? This action cannot be undone.`
            : ''
        }
        confirmText="Delete"
        loading={deleteMutation.isPending}
      />
    ),
    [deleteTarget, confirmDelete, deleteMutation.isPending, entityName, getEntityName]
  );

  return {
    deleteTarget,
    setDeleteTarget,
    confirmDelete,
    isDeleting: deleteMutation.isPending,
    DeleteConfirmModal,
  };
}
