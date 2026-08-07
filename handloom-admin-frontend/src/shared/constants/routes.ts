export const ROUTES = {
  AUTH: {
    LOGIN: '/admin/auth/login',
    LOGOUT: '/admin/auth/logout',
    ME: '/admin/auth/me',
    PASSWORD_CHANGE: '/admin/auth/password/change',
    PASSWORD_RESET_REQUEST: '/admin/auth/password/reset-request',
    PASSWORD_RESET: '/admin/auth/password/reset',
  },

  CATEGORIES: {
    LIST: '/admin/categories',
    DETAIL: (id: string) => `/admin/categories/${id}`,
    ATTRIBUTES: (id: string) => `/admin/categories/${id}/attributes`,
    ATTRIBUTE_DETAIL: (id: string, attrName: string) =>
      `/admin/categories/${id}/attributes/${attrName}`,
  },

  COUPONS: {
    LIST: '/admin/coupons',
    DETAIL: (id: string) => `/admin/coupons/${id}`,
    BY_CODE: (code: string) => `/admin/coupons/code/${code}`,
    VALIDATE: '/admin/coupons/validate',
  },

  CUSTOMERS: {
    LIST: '/admin/customers',
    DETAIL: (id: string) => `/admin/customers/${id}`,
    ORDERS: (id: string) => `/admin/customers/${id}/orders`,
    SEARCH: '/admin/customers/search',
    ADDRESSES: (id: string) => `/admin/customers/${id}/addresses`,
    ADDRESS: (id: string, addressId: string) => `/admin/customers/${id}/addresses/${addressId}`,
  },

  INVENTORY: {
    LOW_STOCK: '/admin/inventory/low-stock',
  },

  NOTIFICATIONS: {
    LIST: '/admin/notifications',
    MY: '/admin/notifications/my',
    MARK_READ: (id: string) => `/admin/notifications/${id}/read`,
    MARK_ALL_READ: '/admin/notifications/read-all',
  },

  ORDERS: {
    LIST: '/admin/orders',
    DETAIL: (id: string) => `/admin/orders/${id}`,
    STATUS: (id: string) => `/admin/orders/${id}/status`,
    NOTES: (id: string) => `/admin/orders/${id}/notes`,
    TRACKING: (id: string) => `/admin/orders/${id}/tracking`,
    CANCEL: (id: string) => `/admin/orders/${id}/cancel`,
    REFUND: (id: string) => `/admin/orders/${id}/refund`,
    PAYMENT_STATUS: (id: string) => `/admin/orders/${id}/payment-status`,
  },

  PRICING: {
    RULES: '/admin/pricing/rules',
    RULE_DETAIL: (id: string) => `/admin/pricing/rules/${id}`,
    RULES_BY_CATEGORY: (categoryId: string) => `/admin/pricing/rules/category/${categoryId}`,
    CALCULATE: '/api/v1/pricing/calculate',
    DIMENSION_OPTIONS: (categoryId: string) => `/api/v1/pricing/dimension-options/${categoryId}`,
  },

  PRODUCTS: {
    LIST: '/admin/products',
    DETAIL: (id: string) => `/admin/products/${id}`,
    INVENTORY: (id: string) => `/admin/products/${id}/inventory`,
    INVENTORY_ADD: (id: string) => `/admin/products/${id}/inventory/add`,
    INVENTORY_REMOVE: (id: string) => `/admin/products/${id}/inventory/remove`,
    INVENTORY_ADJUST: (id: string) => `/admin/products/${id}/inventory/adjust`,
    INVENTORY_TRANSACTIONS: (id: string) => `/admin/products/${id}/inventory/transactions`,
    FILTER_OPTIONS: (categoryId: string) => `/admin/products/filter-options/${categoryId}`,
    REORDER: (categoryId: string) => `/admin/products/categories/${categoryId}/reorder`,
  },

  REPORTS: {
    LIST: '/admin/reports',
    DETAIL: (id: string) => `/admin/reports/${id}`,
    DOWNLOAD: (id: string) => `/admin/reports/${id}/download`,
    SALES: '/admin/reports/sales',
    INVENTORY: '/admin/reports/inventory',
    ORDERS: '/admin/reports/orders',
  },

  SETTINGS: {
    USERS: {
      LIST: '/admin/users',
      DETAIL: (id: string) => `/admin/users/${id}`,
      STATUS: (id: string) => `/admin/users/${id}/status`,
    },
  },
};
