import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { CheckCircle, Clock, Download, FileText, Loader2, Plus, XCircle } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { getErrorMessage, reportsApi } from '../../api';
import {
  Badge,
  Button,
  Card,
  getStatusBadgeVariant,
  Input,
  Modal,
  Pagination,
  Select,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableLoading,
  TableRow,
} from '../../components/common';
import { useCursorPagination } from '../../hooks';
import type { ReportFormat, ReportType } from '../../types';

const REPORT_TYPES: { value: ReportType; label: string; description: string }[] = [
  { value: 'SALES', label: 'Sales Report', description: 'Revenue, orders, and sales trends' },
  { value: 'INVENTORY', label: 'Inventory Report', description: 'Stock levels and movement' },
  { value: 'ORDERS', label: 'Orders Report', description: 'Order details and status' },
  { value: 'CUSTOMERS', label: 'Customers Report', description: 'Customer data and activity' },
  { value: 'PRODUCTS', label: 'Products Report', description: 'Product catalog and performance' },
  { value: 'ARTISANS', label: 'Artisans Report', description: 'Artisan details and earnings' },
];

export function ReportsPage() {
  const queryClient = useQueryClient();
  const { limit, cursor, hasPrevious, goToNextPage, goToPreviousPage, changeLimit } = useCursorPagination(10);
  const [showGenerateModal, setShowGenerateModal] = useState(false);
  const [selectedType, setSelectedType] = useState<ReportType>('SALES');
  const [selectedFormat, setSelectedFormat] = useState<ReportFormat>('CSV');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['reports', { limit, cursor }],
    queryFn: () => reportsApi.list({ limit, cursor }),
  });

  const generateMutation = useMutation({
    mutationFn: (params: {
      type: ReportType;
      format: ReportFormat;
      startDate?: string;
      endDate?: string;
    }) => {
      const filters: Record<string, unknown> = {};
      if (params.startDate) filters.start_date = params.startDate;
      if (params.endDate) filters.end_date = params.endDate;
      return reportsApi.generate(params.type, filters, params.format);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reports'] });
      toast.success('Report generation started');
      setShowGenerateModal(false);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const reports = data?.items || [];
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

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Reports</h1>
          <p className="page-subtitle">Generate and download business reports</p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={() => setShowGenerateModal(true)}>
          Generate Report
        </Button>
      </div>

      {/* Quick Report Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {REPORT_TYPES.map((type) => (
          <Card
            key={type.value}
            className="hover:shadow-md transition-shadow cursor-pointer"
            onClick={() => {
              setSelectedType(type.value);
              setShowGenerateModal(true);
            }}
          >
            <div className="flex items-start gap-4">
              <div className="p-3 bg-primary-50 rounded-lg">
                <FileText className="w-6 h-6 text-primary-600" />
              </div>
              <div>
                <h3 className="font-medium text-gray-900">{type.label}</h3>
                <p className="text-sm text-gray-500">{type.description}</p>
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* Reports History */}
      <Card padding="none">
        <div className="p-6 border-b border-gray-200">
          <h3 className="text-lg font-semibold">Report History</h3>
        </div>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Report</TableHead>
              <TableHead>Format</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={5} colSpan={5} />
            ) : reports.length === 0 ? (
              <TableEmpty colSpan={5} message="No reports generated yet" />
            ) : (
              reports.map((report) => (
                <TableRow key={report.id}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <FileText className="w-5 h-5 text-gray-400" />
                      <span className="font-medium">{report.type} Report</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="gray">{report.format}</Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      {getStatusIcon(report.status)}
                      <Badge variant={getStatusBadgeVariant(report.status)}>{report.status}</Badge>
                    </div>
                  </TableCell>
                  <TableCell>{format(new Date(report.created_at), 'MMM d, yyyy HH:mm')}</TableCell>
                  <TableCell>
                    <div className="flex justify-end">
                      {report.status === 'COMPLETED' && report.file_url && (
                        <Button
                          variant="ghost"
                          size="sm"
                          leftIcon={<Download className="w-4 h-4" />}
                        >
                          Download
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
              itemCount={reports.length}
            />
          </div>
        )}
      </Card>

      {/* Generate Report Modal */}
      <Modal
        isOpen={showGenerateModal}
        onClose={() => setShowGenerateModal(false)}
        title="Generate Report"
        size="md"
      >
        <div className="space-y-4">
          <Select
            label="Report Type"
            options={REPORT_TYPES.map((t) => ({ value: t.value, label: t.label }))}
            value={selectedType}
            onChange={(e) => setSelectedType(e.target.value as ReportType)}
          />
          <Select
            label="Format"
            options={[
              { value: 'CSV', label: 'CSV' },
              { value: 'EXCEL', label: 'Excel' },
              { value: 'PDF', label: 'PDF' },
            ]}
            value={selectedFormat}
            onChange={(e) => setSelectedFormat(e.target.value as ReportFormat)}
          />
          <div className="grid grid-cols-2 gap-4">
            <Input
              label="Start Date"
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
            />
            <Input
              label="End Date"
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
            />
          </div>
          <div className="flex justify-end gap-3 pt-4">
            <Button variant="secondary" onClick={() => setShowGenerateModal(false)}>
              Cancel
            </Button>
            <Button
              onClick={() =>
                generateMutation.mutate({
                  type: selectedType,
                  format: selectedFormat,
                  startDate: startDate || undefined,
                  endDate: endDate || undefined,
                })
              }
              loading={generateMutation.isPending}
            >
              Generate Report
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
