import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import { CheckCircle, Clock, Download, FileText, Loader2, Upload, XCircle } from 'lucide-react';
import { useState } from 'react';

import { bulkApi } from '@/api';
import {
  Badge,
  Button,
  Card,
  getStatusBadgeVariant,
  Pagination,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableLoading,
  TableRow,
} from '@/components/common';
import { useCursorPagination } from '@/hooks';
import type { BulkOperation } from '@/types';

import { BulkExportModal } from './BulkExportModal';
import { BulkImportModal } from './BulkImportModal';

const OPERATION_TYPES = [
  {
    id: 'import-products',
    title: 'Import Products',
    description: 'Bulk import products from CSV/Excel file',
    icon: Upload,
    color: 'bg-blue-100 text-blue-600',
  },
  {
    id: 'update-inventory',
    title: 'Update Inventory',
    description: 'Bulk update stock quantities',
    icon: FileText,
    color: 'bg-green-100 text-green-600',
  },
  {
    id: 'export-products',
    title: 'Export Products',
    description: 'Download product catalog as CSV/Excel',
    icon: Download,
    color: 'bg-purple-100 text-purple-600',
  },
  {
    id: 'export-orders',
    title: 'Export Orders',
    description: 'Download orders data as CSV/Excel',
    icon: Download,
    color: 'bg-orange-100 text-orange-600',
  },
];

type OperationType = 'import-products' | 'update-inventory' | 'export-products' | 'export-orders';

export function BulkOperationsPage() {
  const { limit, cursor, hasPrevious, goToNextPage, goToPreviousPage, changeLimit } =
    useCursorPagination(10);
  const [activeModal, setActiveModal] = useState<OperationType | null>(null);

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['bulk-operations', { limit, cursor }],
    queryFn: () => bulkApi.list({ limit, cursor }),
  });

  const operations = data?.items ?? [];
  const pagination = data?.pagination;

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'COMPLETED':
        return <CheckCircle className="w-4 h-4 text-green-600" />;
      case 'FAILED':
        return <XCircle className="w-4 h-4 text-red-600" />;
      case 'PROCESSING':
        return <Loader2 className="w-4 h-4 text-blue-600 animate-spin" />;
      default:
        return <Clock className="w-4 h-4 text-yellow-600" />;
    }
  };

  const handleOperationClick = (operationId: string) => {
    setActiveModal(operationId as OperationType);
  };

  const handleCloseModal = () => {
    setActiveModal(null);
  };

  const handleDownload = (fileUrl: string) => {
    // If the URL is relative, prepend the API base URL
    const fullUrl = fileUrl.startsWith('http')
      ? fileUrl
      : `${import.meta.env.VITE_API_URL || 'http://localhost:8080'}${fileUrl}`;
    window.open(fullUrl, '_blank');
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="page-title">Bulk Operations</h1>
        <p className="page-subtitle">Import, export, and bulk update your data</p>
      </div>

      {/* Operation Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {OPERATION_TYPES.map((op) => (
          <Card
            key={op.id}
            className="hover:shadow-md transition-shadow cursor-pointer"
            onClick={() => handleOperationClick(op.id)}
          >
            <div className="flex items-start gap-4">
              <div className={`p-3 rounded-lg ${op.color}`}>
                <op.icon className="w-6 h-6" />
              </div>
              <div>
                <h3 className="font-medium text-gray-900">{op.title}</h3>
                <p className="text-sm text-gray-500 mt-1">{op.description}</p>
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* Operations History */}
      <Card padding="none">
        <div className="p-6 border-b border-gray-200 flex items-center justify-between">
          <h3 className="text-lg font-semibold">Operation History</h3>
          <Button variant="ghost" size="sm" onClick={() => refetch()}>
            Refresh
          </Button>
        </div>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Operation</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Progress</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={5} colSpan={6} />
            ) : operations.length === 0 ? (
              <TableEmpty
                colSpan={6}
                message="No bulk operations yet"
                description="Start by selecting an operation above"
              />
            ) : (
              operations
                .map((op) => op as BulkOperation)
                .map((op) => (
                  <TableRow key={op.id}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <FileText className="w-5 h-5 text-gray-400" />
                        <span className="font-medium">{op.entity_type}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="gray">{op.type}</Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <div className="flex-1 h-2 bg-gray-200 rounded-full max-w-[100px]">
                          <div
                            className="h-2 bg-primary-500 rounded-full transition-all"
                            style={{
                              width: `${op.total_records > 0 ? (op.success_count / op.total_records) * 100 : 0}%`,
                            }}
                          />
                        </div>
                        <span className="text-sm text-gray-500">
                          {op.success_count}/{op.total_records}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {getStatusIcon(op.status)}
                        <Badge variant={getStatusBadgeVariant(op.status)}>{op.status}</Badge>
                      </div>
                    </TableCell>
                    <TableCell>
                      {op.created_at ? format(new Date(op.created_at), 'MMM d, yyyy HH:mm') : '-'}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        {op.status === 'COMPLETED' && op.output_file_url && (
                          <Button
                            variant="ghost"
                            size="sm"
                            leftIcon={<Download className="w-4 h-4" />}
                            onClick={() => handleDownload(op.output_file_url)}
                          >
                            Download
                          </Button>
                        )}
                        {op.failure_count > 0 && op.error_file_url && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-red-600"
                            onClick={() => handleDownload(op.error_file_url)}
                          >
                            Errors ({op.failure_count})
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))
            )}
          </TableBody>
        </Table>
        {(pagination?.has_more || hasPrevious) && (
          <div className="border-t border-gray-200 px-6">
            <Pagination
              hasMore={pagination?.has_more ?? false}
              hasPrevious={hasPrevious}
              perPage={limit}
              onNextPage={() => pagination?.next_cursor && goToNextPage(pagination.next_cursor)}
              onPreviousPage={goToPreviousPage}
              onPerPageChange={changeLimit}
              itemCount={operations.length}
            />
          </div>
        )}
      </Card>

      {/* Import Modal */}
      <BulkImportModal
        isOpen={activeModal === 'import-products' || activeModal === 'update-inventory'}
        onClose={handleCloseModal}
        operationType={activeModal === 'update-inventory' ? 'update-inventory' : 'import-products'}
      />

      {/* Export Modal */}
      <BulkExportModal
        isOpen={activeModal === 'export-products' || activeModal === 'export-orders'}
        onClose={handleCloseModal}
        entityType={activeModal === 'export-orders' ? 'orders' : 'products'}
      />
    </div>
  );
}
