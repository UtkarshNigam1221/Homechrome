import apiClient from '@/shared/api/client';
import { ROUTES } from '@/shared/constants/routes';

import type {
  BatchResult,
  CODRemittance,
  CODRemittanceStatus,
  NDRAction,
  Shipment,
  ShippingRate,
  UpdateRateRequest,
} from './types';

export const shippingApi = {
  listRates: async (): Promise<ShippingRate[]> => {
    const response = await apiClient.get<ShippingRate[]>(ROUTES.SHIPPING.RATES);
    return response.data ?? [];
  },

  updateRate: async (
    zone: string,
    slab: number,
    body: UpdateRateRequest
  ): Promise<ShippingRate> => {
    const response = await apiClient.patch<ShippingRate>(
      ROUTES.SHIPPING.RATE_DETAIL(zone, slab),
      body
    );
    return response.data;
  },

  triggerRateRefresh: async (): Promise<{ status: string }> => {
    const response = await apiClient.post<{ status: string }>(ROUTES.SHIPPING.RATES_REFRESH);
    return response.data;
  },

  listCODRemittances: async (status?: CODRemittanceStatus): Promise<CODRemittance[]> => {
    const response = await apiClient.get<CODRemittance[]>(ROUTES.SHIPPING.COD_REMITTANCES, {
      params: status ? { status } : undefined,
    });
    return response.data ?? [];
  },

  getCODRemittance: async (id: string): Promise<CODRemittance> => {
    const response = await apiClient.get<CODRemittance>(ROUTES.SHIPPING.COD_REMITTANCE_DETAIL(id));
    return response.data;
  },

  listNDRQueue: async (): Promise<Shipment[]> => {
    const response = await apiClient.get<Shipment[]>(ROUTES.SHIPPING.NDR_QUEUE);
    return response.data ?? [];
  },

  ndrAction: async (awb: string, action: NDRAction, note?: string): Promise<void> => {
    await apiClient.post(ROUTES.SHIPPING.NDR_ACTION(awb), {
      action: action.toUpperCase(),
      ...(note ? { note } : {}),
    });
  },

  runPickupBatch: async (): Promise<BatchResult> => {
    const response = await apiClient.post<BatchResult>(ROUTES.SHIPPING.PICKUP_BATCH_RUN);
    return response.data;
  },

  listPickupBatches: async (): Promise<BatchResult[]> => {
    const response = await apiClient.get<BatchResult[]>(ROUTES.SHIPPING.PICKUP_BATCHES);
    return response.data ?? [];
  },
};
