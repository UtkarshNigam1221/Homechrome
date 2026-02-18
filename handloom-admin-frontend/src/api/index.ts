import type {
  Artisan,
  AuditLog,
  BulkOperation,
  CalculatePriceRequest,
  CalculatePriceResponse,
  Category,
  CategoryAttribute,
  Coupon,
  CreateArtisanRequest,
  CreateCategoryRequest,
  CreateCouponRequest,
  CreateCustomerRequest,
  CreateOrderRequest,
  CreateProductRequest,
  CreateUserRequest,
  Customer,
  DashboardStats,
  Inventory,
  InventoryTransaction,
  ListResponse,
  Notification,
  Order,
  PaginationParams,
  PricingRule,
  Product,
  Report,
  SalesAnalytics,
  TopCategory,
  TopProduct,
  UploadURLResponse,
  User,
} from '@/types';

import apiClient, { getErrorMessage } from './client';

// Re-export
export { authApi } from './auth';
export { getErrorMessage };

// Normalize backend list responses that may use different keys (e.g. 'products', 'orders')
// into a consistent { items, pagination } shape.
function normalizeListResponse<T>(data: Record<string, unknown>, key: string): ListResponse<T> {
  const items = (data[key] || data.items || data.data || []) as T[];
  const pagination = (data.pagination as ListResponse<T>['pagination']) || {
    limit: 10,
    has_more: false,
  };
  return { items, pagination };
}

// ============================================================================
// Users API
// ============================================================================

export const usersApi = {
  list: async (
    params?: PaginationParams & { role?: string; status?: string; search?: string }
  ): Promise<ListResponse<User>> => {
    const response = await apiClient.get('/admin/users', { params });
    return normalizeListResponse<User>(response.data as Record<string, unknown>, 'users');
  },

  get: async (id: string): Promise<User> => {
    const response = await apiClient.get<User>(`/admin/users/${id}`);
    return response.data;
  },

  create: async (data: CreateUserRequest): Promise<User> => {
    const response = await apiClient.post<User>('/admin/users', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateUserRequest>): Promise<User> => {
    const response = await apiClient.patch<User>(`/admin/users/${id}`, data);
    return response.data;
  },

  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/admin/users/${id}`);
  },

  updateStatus: async (id: string, status: string): Promise<void> => {
    await apiClient.patch(`/admin/users/${id}/status`, { status });
  },
};

// ============================================================================
// Categories API
// ============================================================================
export const categoriesApi = {
  list: async (
    params?: PaginationParams & { status?: string }
  ): Promise<ListResponse<Category>> => {
    const response = await apiClient.get('/admin/categories', { params });
    return normalizeListResponse<Category>(response.data as Record<string, unknown>, 'categories');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Category>(`/admin/categories/${id}`);
    return response.data;
  },

  create: async (data: CreateCategoryRequest) => {
    const response = await apiClient.post<Category>('/admin/categories', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateCategoryRequest>) => {
    const response = await apiClient.patch<Category>(`/admin/categories/${id}`, data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/categories/${id}`);
  },

  addAttribute: async (id: string, attribute: CategoryAttribute) => {
    const response = await apiClient.post<{ attribute: CategoryAttribute }>(
      `/admin/categories/${id}/attributes`,
      attribute
    );
    return response.data;
  },

  updateAttribute: async (id: string, attrName: string, attribute: Partial<CategoryAttribute>) => {
    const response = await apiClient.patch(
      `/admin/categories/${id}/attributes/${attrName}`,
      attribute
    );
    return response.data;
  },

  deleteAttribute: async (id: string, attrName: string) => {
    await apiClient.delete(`/admin/categories/${id}/attributes/${attrName}`);
  },

  getAttributes: async (id: string) => {
    const response = await apiClient.get(`/admin/categories/${id}/attributes`);
    return response.data;
  },
};

// ============================================================================
// Products API
// ============================================================================
export const productsApi = {
  list: async (
    params?: PaginationParams & {
      category_id?: string;
      status?: string;
      min_price?: number;
      max_price?: number;
      in_stock?: boolean;
      low_stock?: boolean;
      search?: string;
      attribute_filters?: Record<string, string[]>;
    }
  ) => {
    // Handle attribute_filters specially - needs to be serialized as JSON
    const { attribute_filters, ...restParams } = params || {};
    const queryParams: Record<string, unknown> = { ...restParams };
    if (attribute_filters && Object.keys(attribute_filters).length > 0) {
      queryParams.attribute_filters = JSON.stringify(attribute_filters);
    }
    const response = await apiClient.get('/admin/products', {
      params: queryParams,
    });
    return normalizeListResponse<Product>(response.data as Record<string, unknown>, 'products');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Product>(`/admin/products/${id}`);
    return response.data;
  },

  create: async (data: CreateProductRequest) => {
    const response = await apiClient.post<Product>('/admin/products', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateProductRequest>) => {
    const response = await apiClient.patch<Product>(`/admin/products/${id}`, data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/products/${id}`);
  },

  getInventory: async (id: string) => {
    const response = await apiClient.get<Inventory>(`/admin/products/${id}/inventory`);
    return response.data;
  },

  addStock: async (id: string, quantity: number, reason?: string) => {
    const response = await apiClient.post(`/admin/products/${id}/inventory/add`, {
      quantity,
      reason,
    });
    return response.data;
  },

  removeStock: async (id: string, quantity: number, reason?: string) => {
    const response = await apiClient.post(`/admin/products/${id}/inventory/remove`, {
      quantity,
      reason,
    });
    return response.data;
  },

  adjustStock: async (id: string, newQuantity: number, reason?: string) => {
    const response = await apiClient.post(`/admin/products/${id}/inventory/adjust`, {
      new_quantity: newQuantity,
      reason,
    });
    return response.data;
  },

  getInventoryTransactions: async (id: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<InventoryTransaction>>(
      `/admin/products/${id}/inventory/transactions`,
      { params }
    );
    return response.data;
  },

  getFilterOptions: async (categoryId: string): Promise<Record<string, string[]>> => {
    const response = await apiClient.get<Record<string, string[]>>(
      `/admin/products/filter-options/${categoryId}`
    );
    return response.data;
  },
};

// ============================================================================
// Inventory API
// ============================================================================
export const inventoryApi = {
  getLowStock: async (params?: PaginationParams): Promise<ListResponse<Inventory>> => {
    const response = await apiClient.get('/admin/inventory/low-stock', {
      params,
    });
    return normalizeListResponse<Inventory>(
      response.data as Record<string, unknown>,
      'inventories'
    );
  },

  addStock: async (productId: string, quantity: number, reason?: string) => {
    await apiClient.post(`/admin/products/${productId}/inventory/add`, { quantity, reason });
  },

  removeStock: async (productId: string, quantity: number, reason?: string) => {
    await apiClient.post(`/admin/products/${productId}/inventory/remove`, { quantity, reason });
  },

  adjustStock: async (productId: string, quantity: number, reason?: string) => {
    await apiClient.post(`/admin/products/${productId}/inventory/adjust`, {
      new_quantity: quantity,
      reason,
    });
  },
};

// ============================================================================
// Orders API
// ============================================================================
export const ordersApi = {
  list: async (
    params?: PaginationParams & {
      status?: string;
      payment_status?: string;
      customer_id?: string;
      start_date?: string;
      end_date?: string;
      search?: string;
    }
  ): Promise<ListResponse<Order>> => {
    const response = await apiClient.get('/admin/orders', { params });
    return normalizeListResponse<Order>(response.data as Record<string, unknown>, 'orders');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Order>(`/admin/orders/${id}`);
    return response.data;
  },

  create: async (data: CreateOrderRequest) => {
    const response = await apiClient.post<Order>('/admin/orders', data);
    return response.data;
  },

  updateStatus: async (id: string, status: string) => {
    await apiClient.patch(`/admin/orders/${id}/status`, { status });
  },

  addNote: async (id: string, note: string, isInternal = true) => {
    await apiClient.post(`/admin/orders/${id}/notes`, { note, is_internal: isInternal });
  },

  updateTracking: async (id: string, trackingNumber: string, carrier?: string) => {
    await apiClient.patch(`/admin/orders/${id}/tracking`, {
      tracking_number: trackingNumber,
      carrier,
    });
  },

  cancel: async (id: string, reason?: string) => {
    await apiClient.post(`/admin/orders/${id}/cancel`, { reason });
  },

  refund: async (id: string, amount: number, reason?: string) => {
    await apiClient.post(`/admin/orders/${id}/refund`, { amount, reason });
  },
};

// ============================================================================
// Customers API
// ============================================================================
export const customersApi = {
  list: async (
    params?: PaginationParams & { status?: string; search?: string }
  ): Promise<ListResponse<Customer>> => {
    const response = await apiClient.get('/admin/customers', { params });
    return normalizeListResponse<Customer>(response.data as Record<string, unknown>, 'customers');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Customer>(`/admin/customers/${id}`);
    return response.data;
  },

  create: async (data: CreateCustomerRequest) => {
    const response = await apiClient.post<Customer>('/admin/customers', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateCustomerRequest>) => {
    const response = await apiClient.put<Customer>(`/admin/customers/${id}`, data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/customers/${id}`);
  },

  getOrders: async (id: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<Order>>(`/admin/customers/${id}/orders`, {
      params,
    });
    return response.data;
  },

  search: async (query: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<Customer>>('/admin/customers/search', {
      params: { q: query, ...params },
    });
    return response.data;
  },
};

// ============================================================================
// Artisans API
// ============================================================================
export const artisansApi = {
  list: async (
    params?: PaginationParams & {
      craft_type?: string;
      location?: string;
      status?: string;
      search?: string;
    }
  ): Promise<ListResponse<Artisan>> => {
    const response = await apiClient.get('/admin/artisans', { params });
    return normalizeListResponse<Artisan>(response.data as Record<string, unknown>, 'artisans');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Artisan>(`/admin/artisans/${id}`);
    return response.data;
  },

  create: async (data: CreateArtisanRequest) => {
    const response = await apiClient.post<Artisan>('/admin/artisans', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateArtisanRequest>) => {
    const response = await apiClient.patch<Artisan>(`/admin/artisans/${id}`, data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/artisans/${id}`);
  },

  updateStatus: async (id: string, status: string) => {
    await apiClient.patch(`/admin/artisans/${id}/status`, { status });
  },

  getProducts: async (id: string, params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<Product>>(`/admin/artisans/${id}/products`, {
      params,
    });
    return response.data;
  },
};

// ============================================================================
// Coupons API
// ============================================================================
export const couponsApi = {
  list: async (
    params?: PaginationParams & {
      status?: string;
      type?: string;
      is_active?: boolean;
      search?: string;
    }
  ): Promise<ListResponse<Coupon>> => {
    const response = await apiClient.get('/admin/coupons', { params });
    return normalizeListResponse<Coupon>(response.data as Record<string, unknown>, 'coupons');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Coupon>(`/admin/coupons/${id}`);
    return response.data;
  },

  getByCode: async (code: string) => {
    const response = await apiClient.get<Coupon>(`/admin/coupons/code/${code}`);
    return response.data;
  },

  create: async (data: CreateCouponRequest) => {
    const response = await apiClient.post<Coupon>('/admin/coupons', data);
    return response.data;
  },

  update: async (id: string, data: Partial<CreateCouponRequest>) => {
    const response = await apiClient.patch<Coupon>(`/admin/coupons/${id}`, data);
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/coupons/${id}`);
  },

  validate: async (
    code: string,
    orderTotal: number,
    customerId?: string,
    productIds?: string[]
  ) => {
    const response = await apiClient.post('/admin/coupons/validate', {
      code,
      order_total: orderTotal,
      customer_id: customerId,
      product_ids: productIds,
    });
    return response.data;
  },
};

// ============================================================================
// Pricing API
// ============================================================================
export const pricingApi = {
  listRules: async (
    params?: PaginationParams & {
      scope_type?: string;
      category_id?: string;
      pricing_type?: string;
      is_active?: boolean;
    }
  ): Promise<ListResponse<PricingRule>> => {
    const response = await apiClient.get('/admin/pricing/rules', {
      params,
    });
    return normalizeListResponse<PricingRule>(response.data as Record<string, unknown>, 'rules');
  },

  getRule: async (id: string) => {
    const response = await apiClient.get<PricingRule>(`/admin/pricing/rules/${id}`);
    return response.data;
  },

  createRule: async (data: Partial<PricingRule>) => {
    const response = await apiClient.post<PricingRule>('/admin/pricing/rules', data);
    return response.data;
  },

  updateRule: async (id: string, data: Partial<PricingRule>) => {
    const response = await apiClient.patch<PricingRule>(`/admin/pricing/rules/${id}`, data);
    return response.data;
  },

  deleteRule: async (id: string) => {
    await apiClient.delete(`/admin/pricing/rules/${id}`);
  },

  getCategoryRules: async (categoryId: string) => {
    const response = await apiClient.get<PricingRule[]>(
      `/admin/pricing/rules/category/${categoryId}`
    );
    return response.data;
  },

  calculatePrice: async (data: CalculatePriceRequest): Promise<CalculatePriceResponse> => {
    const response = await apiClient.post<CalculatePriceResponse>(
      '/api/v1/pricing/calculate',
      data
    );
    return response.data;
  },

  getDimensionOptions: async (categoryId: string) => {
    const response = await apiClient.get(`/api/v1/pricing/dimension-options/${categoryId}`);
    return response.data;
  },
};

// ============================================================================
// Notifications API
// ============================================================================
export const notificationsApi = {
  list: async (
    params?: PaginationParams & { user_id?: string; type?: string; status?: string }
  ): Promise<ListResponse<Notification>> => {
    const response = await apiClient.get('/admin/notifications', {
      params,
    });
    return normalizeListResponse<Notification>(
      response.data as Record<string, unknown>,
      'notifications'
    );
  },

  getMy: async (params?: PaginationParams) => {
    const response = await apiClient.get<ListResponse<Notification>>('/admin/notifications/my', {
      params,
    });
    return response.data;
  },

  send: async (data: {
    user_id: string;
    type: string;
    title: string;
    message: string;
    data?: Record<string, unknown>;
    priority?: string;
  }) => {
    const response = await apiClient.post<Notification>('/admin/notifications', data);
    return response.data;
  },

  markAsRead: async (id: string) => {
    await apiClient.post(`/admin/notifications/${id}/read`);
  },

  markAllAsRead: async () => {
    await apiClient.post('/admin/notifications/read-all');
  },
};

// ============================================================================
// Analytics API
// ============================================================================
export const analyticsApi = {
  getDashboard: async (): Promise<DashboardStats> => {
    const response = await apiClient.get<DashboardStats>('/admin/analytics/dashboard');
    return response.data;
  },

  getSales: async (params?: {
    period?: string;
    start_date?: string;
    end_date?: string;
  }): Promise<SalesAnalytics> => {
    const response = await apiClient.get<SalesAnalytics>('/admin/analytics/sales', { params });
    return response.data;
  },

  getTopProducts: async (params?: {
    limit?: number;
    start_date?: string;
    end_date?: string;
  }): Promise<TopProduct[]> => {
    const response = await apiClient.get('/admin/analytics/top-products', { params });
    const data = response.data;
    return Array.isArray(data)
      ? data
      : ((data as Record<string, unknown>).products as TopProduct[]) || [];
  },

  getTopCategories: async (params?: {
    limit?: number;
    start_date?: string;
    end_date?: string;
  }): Promise<TopCategory[]> => {
    const response = await apiClient.get('/admin/analytics/top-categories', { params });
    const data = response.data;
    return Array.isArray(data)
      ? data
      : ((data as Record<string, unknown>).categories as TopCategory[]) || [];
  },

  getCustomerAnalytics: async (params?: { start_date?: string; end_date?: string }) => {
    const response = await apiClient.get('/admin/analytics/customers', { params });
    return response.data;
  },

  getInventoryAnalytics: async () => {
    const response = await apiClient.get('/admin/analytics/inventory');
    return response.data;
  },
};

// ============================================================================
// Bulk Operations API
// ============================================================================
export const bulkApi = {
  list: async (
    params?: PaginationParams & { type?: string; entity_type?: string; status?: string }
  ): Promise<ListResponse<BulkOperation>> => {
    const response = await apiClient.get('/admin/bulk', { params });
    return normalizeListResponse<BulkOperation>(
      response.data as Record<string, unknown>,
      'operations'
    );
  },

  get: async (id: string) => {
    const response = await apiClient.get<BulkOperation>(`/admin/bulk/${id}`);
    return response.data;
  },

  importProducts: async (fileUrl: string, mapping?: Record<string, string>) => {
    const response = await apiClient.post<BulkOperation>('/admin/bulk/products/import', {
      file_url: fileUrl,
      mapping,
    });
    return response.data;
  },

  updateInventory: async (
    fileUrl?: string,
    updates?: { product_id: string; quantity: number }[]
  ) => {
    const response = await apiClient.post<BulkOperation>('/admin/bulk/inventory/update', {
      file_url: fileUrl,
      updates,
    });
    return response.data;
  },

  exportData: async (entityType: string, filters?: Record<string, unknown>, format = 'CSV') => {
    const response = await apiClient.post<BulkOperation>('/admin/bulk/export', {
      entity_type: entityType,
      filters,
      format,
    });
    return response.data;
  },

  cancel: async (id: string) => {
    await apiClient.post(`/admin/bulk/${id}/cancel`);
  },

  getUploadUrl: async (entityType: string, filename: string) => {
    const response = await apiClient.post('/admin/bulk/upload-url', {
      entity_type: entityType,
      filename,
    });
    return response.data;
  },

  getDownloadUrl: async (id: string) => {
    const response = await apiClient.get(`/admin/bulk/${id}/download`);
    return response.data;
  },
};

// ============================================================================
// Reports API
// ============================================================================
export const reportsApi = {
  list: async (
    params?: PaginationParams & {
      type?: string;
      status?: string;
      start_date?: string;
      end_date?: string;
    }
  ): Promise<ListResponse<Report>> => {
    const response = await apiClient.get('/admin/reports', { params });
    return normalizeListResponse<Report>(response.data as Record<string, unknown>, 'reports');
  },

  get: async (id: string) => {
    const response = await apiClient.get<Report>(`/admin/reports/${id}`);
    return response.data;
  },

  generate: async (type: string, filters?: Record<string, unknown>, format = 'CSV') => {
    const response = await apiClient.post<Report>('/admin/reports', {
      type,
      filters,
      format,
    });
    return response.data;
  },

  delete: async (id: string) => {
    await apiClient.delete(`/admin/reports/${id}`);
  },

  getDownloadUrl: async (id: string) => {
    const response = await apiClient.get(`/admin/reports/${id}/download`);
    return response.data;
  },

  generateSalesReport: async (startDate: string, endDate: string, format = 'CSV') => {
    const response = await apiClient.post<Report>('/admin/reports/sales', {
      start_date: startDate,
      end_date: endDate,
      format,
    });
    return response.data;
  },

  generateInventoryReport: async (format = 'CSV') => {
    const response = await apiClient.post<Report>('/admin/reports/inventory', { format });
    return response.data;
  },

  generateOrdersReport: async (
    startDate: string,
    endDate: string,
    status?: string,
    format = 'CSV'
  ) => {
    const response = await apiClient.post<Report>('/admin/reports/orders', {
      start_date: startDate,
      end_date: endDate,
      status,
      format,
    });
    return response.data;
  },
};

// ============================================================================
// Assets API (tmp/ → assets/ S3 flow)
// ============================================================================
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

// ============================================================================
// Audit API
// ============================================================================
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
