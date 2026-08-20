// Helper function to get badge variant from status
export function getStatusBadgeVariant(
  status: string
): 'success' | 'warning' | 'danger' | 'info' | 'gray' {
  const statusMap: Record<string, 'success' | 'warning' | 'danger' | 'info' | 'gray'> = {
    // Common statuses
    ACTIVE: 'success',
    INACTIVE: 'gray',
    PENDING: 'warning',
    PROCESSING: 'info',
    COMPLETED: 'success',
    FAILED: 'danger',
    CANCELLED: 'danger',

    // Order statuses
    CONFIRMED: 'info',
    SHIPPED: 'info',
    DELIVERED: 'success',
    RETURNED: 'warning',

    // Payment statuses
    PAID: 'success',
    REFUNDED: 'warning',
    PARTIALLY_REFUNDED: 'warning',

    // Product statuses
    DRAFT: 'gray',

    // Customer statuses
    BLOCKED: 'danger',

    // Notification statuses
    UNREAD: 'info',
    READ: 'gray',
    ARCHIVED: 'gray',

    // Coupon statuses
    EXPIRED: 'danger',
  };

  if (!status) return 'gray';
  return statusMap[status.toUpperCase()] || 'gray';
}
