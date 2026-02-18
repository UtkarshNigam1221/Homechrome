import apiClient from '@/shared/api/client';
import type { UploadURLResponse } from '@/shared/types/common';

export const assetsApi = {
  getUploadUrl: async (
    fileName: string,
    type: 'IMAGE' | 'VIDEO' | 'DOCUMENT',
    contentType: string,
    size: number
  ): Promise<UploadURLResponse> => {
    const response = await apiClient.post<UploadURLResponse>('/admin/assets/upload-url', {
      file_name: fileName,
      content_type: contentType,
      size,
      type,
    });
    return response.data;
  },

  delete: async (url: string) => {
    await apiClient.delete('/admin/assets', {
      data: { url },
    });
  },
};
