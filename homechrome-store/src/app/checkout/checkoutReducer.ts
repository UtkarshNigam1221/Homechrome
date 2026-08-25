import { CartWithItems } from '@/types';

export type CheckoutStep = 'address' | 'review';

export interface CheckoutState {
  cart: CartWithItems | null;
  cartLoading: boolean;
  step: CheckoutStep;
  selectedAddressId: string | null;
  showAddressForm: boolean;
  addressSaving: boolean;
  initiating: boolean;
  error: string | null;
  couponCode: string | null;
  couponDiscount: number;
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
  | { type: 'PAYMENT_START' }
  | { type: 'PAYMENT_FAILED'; error: string }
  | { type: 'CLEAR_ERROR' }
  | { type: 'COUPON_APPLIED'; code: string; discount: number }
  | { type: 'COUPON_REMOVED' };

export const initialCheckoutState: CheckoutState = {
  cart: null,
  cartLoading: true,
  step: 'address',
  selectedAddressId: null,
  showAddressForm: false,
  addressSaving: false,
  initiating: false,
  error: null,
  couponCode: null,
  couponDiscount: 0,
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
    case 'PAYMENT_START':
      return { ...state, initiating: true, error: null };
    case 'PAYMENT_FAILED':
      return { ...state, initiating: false, error: action.error };
    case 'CLEAR_ERROR':
      return { ...state, error: null };
    case 'COUPON_APPLIED':
      return { ...state, couponCode: action.code, couponDiscount: action.discount };
    case 'COUPON_REMOVED':
      return { ...state, couponCode: null, couponDiscount: 0 };
    default:
      return state;
  }
}
