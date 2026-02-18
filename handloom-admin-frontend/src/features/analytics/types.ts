export interface DashboardStats {
  total_revenue: number;
  total_orders: number;
  total_customers: number;
  active_products: number;
  low_stock_count: number;
  pending_orders: number;
  revenue_change?: number;
  orders_change?: number;
}

export interface SalesAnalytics {
  period: string;
  data: {
    date: string;
    revenue: number;
    orders: number;
    average_order_value: number;
  }[];
  totals: {
    revenue: number;
    orders: number;
    average_order_value: number;
  };
}

export interface TopProduct {
  product_id: string;
  product_name: string;
  sku: string;
  quantity_sold: number;
  revenue: number;
  image_url?: string;
}

export interface TopCategory {
  category_id: string;
  category_name: string;
  product_count: number;
  revenue: number;
}
