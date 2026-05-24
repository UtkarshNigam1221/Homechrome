import { CartWithItems, CourierOption } from '@/types';

export type CheckoutStep = 'address' | 'shipping' | 'review';

export interface CheckoutState {
  cart: CartWithItems | null;
  cartLoading: boolean;
  step: CheckoutStep;
  selectedAddressId: string | null;
  showAddressForm: boolean;
  addressSaving: boolean;
  couriers: CourierOption[];
  selectedCourierId: number | null;
  serviceabilityLoading: boolean;
  serviceabilityError: string | null;
  initiating: boolean;
  error: string | null;
}

export type CheckoutAction =
  | { type: 'CART_LOADING' }
  | { type: 'CART_LOADED'; cart: CartWithItems }
  | { type: 'GO_TO_STEP'; step: CheckoutStep }
  | { type: 'SELECT_ADDRESS'; id: string }
  | { type: 'TOGGLE_ADDRESS_FORM'; show: boolean }
  | { type: 'ADDRESS_SAVE_START' }
  | { type: 'ADDRESS_SAVED'; addressId: string }
  | { type: 'ADDRESS_SAVE_FAILED'; error: string }
  | { type: 'SERVICEABILITY_START' }
  | { type: 'SERVICEABILITY_SUCCESS'; couriers: CourierOption[] }
  | { type: 'SERVICEABILITY_FAIL'; error: string }
  | { type: 'SELECT_COURIER'; id: number }
  | { type: 'PAYMENT_START' }
  | { type: 'PAYMENT_FAILED'; error: string }
  | { type: 'CLEAR_ERROR' };

export const initialCheckoutState: CheckoutState = {
  cart: null,
  cartLoading: true,
  step: 'address',
  selectedAddressId: null,
  showAddressForm: false,
  addressSaving: false,
  couriers: [],
  selectedCourierId: null,
  serviceabilityLoading: false,
  serviceabilityError: null,
  initiating: false,
  error: null,
};

export function checkoutReducer(state: CheckoutState, action: CheckoutAction): CheckoutState {
  switch (action.type) {
    case 'CART_LOADING':
      return { ...state, cartLoading: true };
    case 'CART_LOADED':
      return { ...state, cart: action.cart, cartLoading: false };
    case 'GO_TO_STEP':
      return { ...state, step: action.step, error: null };
    case 'SELECT_ADDRESS':
      return { ...state, selectedAddressId: action.id };
    case 'TOGGLE_ADDRESS_FORM':
      return { ...state, showAddressForm: action.show };
    case 'ADDRESS_SAVE_START':
      return { ...state, addressSaving: true, error: null };
    case 'ADDRESS_SAVED':
      return {
        ...state,
        addressSaving: false,
        showAddressForm: false,
        selectedAddressId: action.addressId,
      };
    case 'ADDRESS_SAVE_FAILED':
      return { ...state, addressSaving: false, error: action.error };
    case 'SERVICEABILITY_START':
      return {
        ...state,
        serviceabilityLoading: true,
        serviceabilityError: null,
        couriers: [],
        selectedCourierId: null,
      };
    case 'SERVICEABILITY_SUCCESS': {
      const autoSelect = action.couriers.length === 1 ? action.couriers[0].id : null;
      return {
        ...state,
        serviceabilityLoading: false,
        couriers: action.couriers,
        selectedCourierId: autoSelect,
      };
    }
    case 'SERVICEABILITY_FAIL':
      return {
        ...state,
        serviceabilityLoading: false,
        serviceabilityError: action.error,
      };
    case 'SELECT_COURIER':
      return { ...state, selectedCourierId: action.id };
    case 'PAYMENT_START':
      return { ...state, initiating: true, error: null };
    case 'PAYMENT_FAILED':
      return { ...state, initiating: false, error: action.error };
    case 'CLEAR_ERROR':
      return { ...state, error: null };
    default:
      return state;
  }
}
