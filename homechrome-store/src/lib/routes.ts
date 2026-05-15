export const ROUTES = {
  AUTH: {
    SEND_OTP: '/api/v1/store/auth/otp/send',
    VERIFY_OTP: '/api/v1/store/auth/otp/verify',
    LOGOUT: '/api/v1/store/auth/logout',
    REFRESH: '/api/v1/store/auth/refresh',
  },
  ME: {
    PROFILE: '/api/v1/store/me',
    ADDRESSES: '/api/v1/store/me/addresses',
    ADDRESS: (id: string) => `/api/v1/store/me/addresses/${id}`,
  },
  CART: {
    ROOT: '/api/v1/store/cart',
    ITEMS: '/api/v1/store/cart/items',
    ITEM: (productId: string) => `/api/v1/store/cart/items/${productId}`,
  },
  CHECKOUT: {
    SERVICEABILITY: '/api/v1/store/checkout/serviceability',
    INITIATE: '/api/v1/store/checkout/initiate',
    PAYMENT_STATUS: (orderId: string) => `/api/v1/store/checkout/payment-status/${orderId}`,
  },
  ORDERS: {
    LIST: '/api/v1/store/orders',
    DETAIL: (orderId: string) => `/api/v1/store/orders/${orderId}`,
    CANCEL: (orderId: string) => `/api/v1/store/orders/${orderId}/cancel`,
  },
  CATALOG: {
    CATEGORIES: '/api/v1/store/catalog/categories',
    CATEGORY: (slug: string) => `/api/v1/store/catalog/categories/${slug}`,
    PRODUCTS: '/api/v1/store/catalog/products',
    PRODUCT: (slug: string) => `/api/v1/store/catalog/products/${slug}`,
    FILTER_OPTIONS: (categoryId: string) => `/api/v1/store/catalog/products/filter-options/${categoryId}`,
  },
  SHIPPING: {
    CHECK_PINCODE: (pin: string) => `/api/v1/store/catalog/check-pincode/${pin}`,
  },
  TRACK: (trackingNumber: string) => `/api/v1/store/track/${encodeURIComponent(trackingNumber)}`,
  EVENTS: '/api/v1/store/events',
} as const;
