import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import type { ServiceabilityResult } from '@/types';

export const shippingApi = {
  checkPincode: async (pincode: string): Promise<ServiceabilityResult> => {
    const { data } = await api.get<ServiceabilityResult>(
      ROUTES.SHIPPING.CHECK_PINCODE(pincode),
    );
    return data;
  },
};
