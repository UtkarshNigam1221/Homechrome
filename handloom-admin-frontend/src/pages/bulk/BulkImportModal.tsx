import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, CheckCircle, FileText, Upload, X } from 'lucide-react';
import { useCallback, useRef, useState } from 'react';
import toast from 'react-hot-toast';

import { bulkApi, getErrorMessage } from '../../api';
import apiClient from '../../api/client';
import { Button, Modal } from '../../components/common';

interface BulkImportModalProps {
  isOpen: boolean;
  onClose: () => void;
  operationType: 'import-products' | 'update-inventory';
}

export function BulkImportModal({ isOpen, onClose, operationType }: BulkImportModalProps) {
  const queryClient = useQueryClient();
  const [file, setFile] = useState<File | null>(null);
  const [uploadedFileUrl, setUploadedFileUrl] = useState<string>('');
  const [isUploading, setIsUploading] = useState(false);
  const [dragActive, setDragActive] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFile = useCallback(async (selectedFile: File) => {
    // Validate file type
    const validTypes = [
      'text/csv',
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      'application/vnd.ms-excel',
    ];
    const validExtensions = ['.csv', '.xlsx', '.xls'];
    const fileExtension = selectedFile.name.toLowerCase().slice(selectedFile.name.lastIndexOf('.'));

    if (!validTypes.includes(selectedFile.type) && !validExtensions.includes(fileExtension)) {
      toast.error('Please upload a CSV or Excel file');
      return;
    }

    setFile(selectedFile);
    setIsUploading(true);

    try {
      // Upload file to get URL
      const formData = new FormData();
      formData.append('file', selectedFile);

      const response = await apiClient.post<{ url: string }>('/uploads', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });

      setUploadedFileUrl(response.data.url);
      toast.success('File uploaded successfully');
    } catch (_error) {
      toast.error('Failed to upload file');
      setFile(null);
    } finally {
      setIsUploading(false);
    }
  }, []);

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setDragActive(false);
      if (e.dataTransfer.files && e.dataTransfer.files[0]) {
        handleFile(e.dataTransfer.files[0]);
      }
    },
    [handleFile]
  );

  const handleClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      handleFile(e.target.files[0]);
    }
  };

  // Import products mutation
  const importProductsMutation = useMutation({
    mutationFn: (fileUrl: string) => bulkApi.importProducts(fileUrl),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bulk-operations'] });
      toast.success('Import started successfully');
      handleClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Update inventory mutation
  const updateInventoryMutation = useMutation({
    mutationFn: (fileUrl: string) => bulkApi.updateInventory(fileUrl),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bulk-operations'] });
      toast.success('Inventory update started successfully');
      handleClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const handleSubmit = () => {
    if (!uploadedFileUrl) {
      toast.error('Please upload a file first');
      return;
    }

    if (operationType === 'import-products') {
      importProductsMutation.mutate(uploadedFileUrl);
    } else {
      updateInventoryMutation.mutate(uploadedFileUrl);
    }
  };

  const handleClose = () => {
    setFile(null);
    setUploadedFileUrl('');
    onClose();
  };

  const handleRemoveFile = (e: React.MouseEvent) => {
    e.stopPropagation();
    setFile(null);
    setUploadedFileUrl('');
  };

  const isLoading =
    importProductsMutation.isPending || updateInventoryMutation.isPending || isUploading;

  const title = operationType === 'import-products' ? 'Import Products' : 'Update Inventory';
  const description =
    operationType === 'import-products'
      ? 'Upload a CSV or Excel file containing product data to import.'
      : 'Upload a CSV or Excel file containing inventory updates.';

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title={title} size="md">
      <div className="space-y-6">
        <p className="text-sm text-gray-600">{description}</p>

        {/* File dropzone */}
        <div
          className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
            dragActive
              ? 'border-primary-500 bg-primary-50'
              : file
                ? 'border-green-500 bg-green-50'
                : 'border-gray-300 hover:border-primary-400'
          }`}
          onDragEnter={handleDrag}
          onDragLeave={handleDrag}
          onDragOver={handleDrag}
          onDrop={handleDrop}
          onClick={handleClick}
        >
          <input
            ref={fileInputRef}
            type="file"
            accept=".csv,.xlsx,.xls"
            onChange={handleFileChange}
            className="hidden"
          />
          {file ? (
            <div className="flex flex-col items-center gap-2">
              <CheckCircle className="w-10 h-10 text-green-600" />
              <p className="font-medium text-gray-900">{file.name}</p>
              <p className="text-sm text-gray-500">{(file.size / 1024).toFixed(1)} KB</p>
              <Button variant="ghost" size="sm" onClick={handleRemoveFile}>
                <X className="w-4 h-4 mr-1" /> Remove
              </Button>
            </div>
          ) : isUploading ? (
            <div className="flex flex-col items-center gap-2">
              <div className="animate-spin rounded-full h-10 w-10 border-b-2 border-primary-600" />
              <p className="text-gray-600">Uploading...</p>
            </div>
          ) : (
            <div className="flex flex-col items-center gap-2">
              <Upload className="w-10 h-10 text-gray-400" />
              <p className="text-gray-600">
                {dragActive
                  ? 'Drop the file here...'
                  : 'Drag & drop a file here, or click to select'}
              </p>
              <p className="text-sm text-gray-400">CSV or Excel files only</p>
            </div>
          )}
        </div>

        {/* File format instructions */}
        <div className="bg-gray-50 rounded-lg p-4">
          <h4 className="font-medium text-gray-900 mb-2 flex items-center gap-2">
            <FileText className="w-4 h-4" />
            File Format
          </h4>
          {operationType === 'import-products' ? (
            <div className="text-sm text-gray-600 space-y-2">
              <p>
                Required columns: <code className="bg-gray-200 px-1 rounded">sku</code>,{' '}
                <code className="bg-gray-200 px-1 rounded">name</code>,{' '}
                <code className="bg-gray-200 px-1 rounded">category_id</code>,{' '}
                <code className="bg-gray-200 px-1 rounded">base_price</code>,{' '}
                <code className="bg-gray-200 px-1 rounded">selling_price</code>
              </p>
              <p>
                Optional columns: <code className="bg-gray-200 px-1 rounded">description</code>,{' '}
                <code className="bg-gray-200 px-1 rounded">design_id</code>,{' '}
                <code className="bg-gray-200 px-1 rounded">quantity</code>,{' '}
                <code className="bg-gray-200 px-1 rounded">weight</code>,{' '}
                <code className="bg-gray-200 px-1 rounded">tags</code>
              </p>
              <p className="pt-1">
                <a
                  href="/samples/product_import_sample.csv"
                  download
                  className="text-primary-600 hover:underline"
                >
                  Download sample CSV
                </a>
              </p>
            </div>
          ) : (
            <div className="text-sm text-gray-600 space-y-2">
              <p>
                Required columns: <code className="bg-gray-200 px-1 rounded">product_id</code> or{' '}
                <code className="bg-gray-200 px-1 rounded">sku</code>,{' '}
                <code className="bg-gray-200 px-1 rounded">quantity</code>
              </p>
              <p>
                Optional columns: <code className="bg-gray-200 px-1 rounded">operation</code> (SET,
                ADD, SUBTRACT), <code className="bg-gray-200 px-1 rounded">reason</code>
              </p>
              <p className="pt-1">
                <a
                  href="/samples/inventory_update_sample.csv"
                  download
                  className="text-primary-600 hover:underline"
                >
                  Download sample CSV
                </a>
              </p>
            </div>
          )}
        </div>

        {/* Warning */}
        <div className="flex items-start gap-3 p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
          <AlertCircle className="w-5 h-5 text-yellow-600 flex-shrink-0 mt-0.5" />
          <div className="text-sm text-yellow-800">
            <p className="font-medium">Important</p>
            <p>
              This operation will process records in the background. You can track progress in the
              operation history below.
            </p>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={handleClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} loading={isLoading} disabled={!uploadedFileUrl}>
            Start Import
          </Button>
        </div>
      </div>
    </Modal>
  );
}
