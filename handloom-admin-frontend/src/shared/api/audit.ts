import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { AuditLog, ListResponse, PaginationParams } from '@/shared/types/common';

export const auditApi = {
  list: async (
    params?: PaginationParams & {
      action?: string;
      entity_type?: string;
      entity_id?: string;
      user_id?: string;
    }
  ): Promise<ListResponse<AuditLog>> => {
    const response = await apiClient.get('/admin/audit', { params });
    return normalizeListResponse<AuditLog>(response.data as Record<string, unknown>, 'logs');
  },

  get: async (id: string) => {
    const response = await apiClient.get<AuditLog>(`/admin/audit/${id}`);
    return response.data;
  },

  getByEntity: async (entityType: string, entityId: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<AuditLog>>(
      `/admin/audit/entity/${entityType}/${entityId}`,
      { params }
    );
    return response.data;
  },

  getByUser: async (userId: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<AuditLog>>(`/admin/audit/user/${userId}`, {
      params,
    });
    return response.data;
  },
};
