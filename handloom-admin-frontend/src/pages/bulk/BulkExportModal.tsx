import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, Download } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { bulkApi, getErrorMessage } from '../../api';
import { Button, Modal, Select } from '../../components/common';

interface BulkExportModalProps {
  isOpen: boolean;
  onClose: () => void;
  entityType: 'products' | 'orders';
}

export function BulkExportModal({ isOpen, onClose, entityType }: BulkExportModalProps) {
  const queryClient = useQueryClient();
  const [format, setFormat] = useState<'CSV' | 'JSON'>('CSV');
  const [statusFilter, setStatusFilter] = useState<string>('');

  // Export mutation
  const exportMutation = useMutation({
    mutationFn: () => {
      const filters: Record<string, unknown> = {};
      if (statusFilter) {
        filters.status = statusFilter;
      }
      // Backend expects singular form: PRODUCT, ORDER (not PRODUCTS, ORDERS)
      const backendEntityType = entityType === 'products' ? 'PRODUCT' : 'ORDER';
      return bulkApi.exportData(backendEntityType, filters, format);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bulk-operations'] });
      toast.success('Export started successfully. Check operation history for download link.');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const handleSubmit = () => {
    exportMutation.mutate();
  };

  const title = entityType === 'products' ? 'Export Products' : 'Export Orders';
  const description =
    entityType === 'products'
      ? 'Export your product catalog to a file.'
      : 'Export your orders data to a file.';

  const statusOptions =
    entityType === 'products'
      ? [
          { value: '', label: 'All Statuses' },
          { value: 'ACTIVE', label: 'Active' },
          { value: 'INACTIVE', label: 'Inactive' },
          { value: 'DRAFT', label: 'Draft' },
        ]
      : [
          { value: '', label: 'All Statuses' },
          { value: 'PENDING', label: 'Pending' },
          { value: 'CONFIRMED', label: 'Confirmed' },
          { value: 'PROCESSING', label: 'Processing' },
          { value: 'SHIPPED', label: 'Shipped' },
          { value: 'DELIVERED', label: 'Delivered' },
          { value: 'CANCELLED', label: 'Cancelled' },
        ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={title} size="md">
      <div className="space-y-6">
        <p className="text-sm text-gray-600">{description}</p>

        {/* Export icon */}
        <div className="flex justify-center py-4">
          <div className="p-4 bg-purple-100 rounded-full">
            <Download className="w-12 h-12 text-purple-600" />
          </div>
        </div>

        {/* Options */}
        <div className="space-y-4">
          <Select
            label="Export Format"
            value={format}
            onChange={(e) => setFormat(e.target.value as 'CSV' | 'JSON')}
            options={[
              { value: 'CSV', label: 'CSV (Comma Separated Values)' },
              { value: 'JSON', label: 'JSON (JavaScript Object Notation)' },
            ]}
          />

          <Select
            label="Filter by Status"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            options={statusOptions}
          />
        </div>

        {/* What will be exported */}
        <div className="bg-gray-50 rounded-lg p-4">
          <h4 className="font-medium text-gray-900 mb-2">What will be exported:</h4>
          {entityType === 'products' ? (
            <ul className="text-sm text-gray-600 space-y-1 list-disc list-inside">
              <li>Product ID, SKU, Name</li>
              <li>Category, Design, Description</li>
              <li>Base Price, Selling Price</li>
              <li>Quantity, Available Stock</li>
              <li>Status, Tags, Metadata</li>
            </ul>
          ) : (
            <ul className="text-sm text-gray-600 space-y-1 list-disc list-inside">
              <li>Order ID, Order Number</li>
              <li>Customer Name, Email, Phone</li>
              <li>Items, Quantities, Prices</li>
              <li>Shipping Address</li>
              <li>Order Status, Payment Status</li>
              <li>Created Date, Updated Date</li>
            </ul>
          )}
        </div>

        {/* Info */}
        <div className="flex items-start gap-3 p-3 bg-blue-50 border border-blue-200 rounded-lg">
          <AlertCircle className="w-5 h-5 text-blue-600 flex-shrink-0 mt-0.5" />
          <div className="text-sm text-blue-800">
            <p>
              The export will be processed in the background. Once complete, you can download the
              file from the operation history.
            </p>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={onClose} disabled={exportMutation.isPending}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            loading={exportMutation.isPending}
            leftIcon={<Download className="w-4 h-4" />}
          >
            Start Export
          </Button>
        </div>
      </div>
    </Modal>
  );
}
