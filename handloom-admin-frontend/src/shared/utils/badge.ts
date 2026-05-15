// Helper function to get badge variant from status.
// Terminal states: CANCELLED = gray (neutral), REFUNDED = success (positive resolution).
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
    CANCELLED: 'gray',

    // Order statuses
    CONFIRMED: 'info',
    SHIPPED: 'info',
    DELIVERED: 'success',

    // Payment statuses
    PAID: 'success',
    REFUNDED: 'success',

    // Product statuses
    DRAFT: 'gray',

    // User statuses
    SUSPENDED: 'danger',

    // Notification statuses
    UNREAD: 'info',
    READ: 'gray',
    ARCHIVED: 'gray',

    // Coupon statuses
    EXPIRED: 'danger',

    // Shipping / returns / reconciliation
    RECEIVED: 'info',
    RECONCILED: 'success',
    UNMATCHED: 'danger',
    NDR: 'warning',
    NDR_ESCALATED: 'danger',
    MANIFESTED: 'info',
    OUT_FOR_DELIVERY: 'info',
    IN_TRANSIT: 'info',
    PICKED_UP: 'info',
    RTO: 'danger',
    RETURNING: 'warning',
    RETURNED: 'success',
    REQUESTED: 'warning',
  };

  if (!status) return 'gray';
  return statusMap[status.toUpperCase()] || 'gray';
}
