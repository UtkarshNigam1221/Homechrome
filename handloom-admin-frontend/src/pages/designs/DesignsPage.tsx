import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Edit, Image, Palette, Plus, Search, Trash2 } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { designsApi, getErrorMessage } from '../../api';
import {
  Badge,
  Button,
  Card,
  ConfirmModal,
  getStatusBadgeVariant,
  Input,
  Pagination,
} from '../../components/common';
import type { Design } from '../../types';
import { DesignFormModal } from './DesignFormModal';

export function DesignsPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(12);
  const [searchQuery, setSearchQuery] = useState('');
  const [showFormModal, setShowFormModal] = useState(false);
  const [editingDesign, setEditingDesign] = useState<Design | null>(null);
  const [deleteDesign, setDeleteDesign] = useState<Design | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['designs', { page, limit: perPage, search: searchQuery }],
    queryFn: () => designsApi.list({ page, limit: perPage, search: searchQuery || undefined }),
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: designsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['designs'] });
      toast.success('Design deleted successfully');
      setDeleteDesign(null);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Handle various response formats from the API
  // Backend returns { designs: [...], pagination: {...} }
  const extractDesigns = (responseData: unknown): Design[] => {
    if (!responseData) return [];
    if (Array.isArray(responseData)) return responseData as Design[];
    if (typeof responseData === 'object' && responseData !== null) {
      const record = responseData as Record<string, unknown>;
      if ('designs' in record) return record.designs as Design[];
      if ('items' in record) return record.items as Design[];
      if ('data' in record) return Array.isArray(record.data) ? (record.data as Design[]) : [];
    }
    return [];
  };

  const designs = extractDesigns(data);
  const pagination = data?.pagination;

  const handleOpenCreate = () => {
    setEditingDesign(null);
    setShowFormModal(true);
  };

  const handleEdit = (design: Design) => {
    setEditingDesign(design);
    setShowFormModal(true);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Designs</h1>
          <p className="page-subtitle">Manage product designs and patterns</p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
          Add Design
        </Button>
      </div>

      <Card padding="sm">
        <Input
          placeholder="Search designs..."
          value={searchQuery}
          onChange={(e) => {
            setSearchQuery(e.target.value);
            setPage(1);
          }}
          leftIcon={<Search className="w-4 h-4" />}
        />
      </Card>

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {Array.from({ length: 8 }).map((_, i) => (
            <Card key={i} className="animate-pulse">
              <div className="aspect-square bg-gray-200 rounded-lg mb-4" />
              <div className="h-4 bg-gray-200 rounded w-3/4 mb-2" />
              <div className="h-3 bg-gray-200 rounded w-1/2" />
            </Card>
          ))}
        </div>
      ) : designs.length === 0 ? (
        <Card className="text-center py-12">
          <Palette className="w-12 h-12 mx-auto text-gray-300 mb-4" />
          <p className="text-gray-500">No designs found</p>
          <Button
            className="mt-4"
            leftIcon={<Plus className="w-4 h-4" />}
            onClick={handleOpenCreate}
          >
            Create your first design
          </Button>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {designs.map((design) => (
            <Card
              key={design.id}
              className="overflow-hidden hover:shadow-md transition-shadow group"
              padding="none"
            >
              <div
                className="aspect-square bg-gray-100 flex items-center justify-center relative cursor-pointer"
                onClick={() => handleEdit(design)}
              >
                {design.images && design.images.length > 0 ? (
                  <img
                    src={design.images[0].url}
                    alt={design.images[0].alt_text || design.name}
                    className="w-full h-full object-cover"
                  />
                ) : design.preview_image_url ? (
                  <img
                    src={design.preview_image_url}
                    alt={design.name}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <Image className="w-12 h-12 text-gray-300" />
                )}
                {/* Overlay with actions */}
                <div className="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleEdit(design);
                    }}
                  >
                    <Edit className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="danger"
                    size="sm"
                    onClick={(e) => {
                      e.stopPropagation();
                      setDeleteDesign(design);
                    }}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </div>
              </div>
              <div className="p-4">
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <h3 className="font-medium text-gray-900">{design.name}</h3>
                    <p className="text-sm text-gray-500">{design.product_count || 0} products</p>
                  </div>
                  <Badge variant={getStatusBadgeVariant(design.status)} size="sm">
                    {design.status}
                  </Badge>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {pagination && pagination.total_pages > 1 && (
        <Pagination
          currentPage={pagination.current_page}
          totalPages={pagination.total_pages}
          totalCount={pagination.total_count}
          perPage={pagination.per_page}
          onPageChange={setPage}
          onPerPageChange={(v) => {
            setPerPage(v);
            setPage(1);
          }}
        />
      )}

      {/* Design Form Modal */}
      <DesignFormModal
        isOpen={showFormModal}
        onClose={() => {
          setShowFormModal(false);
          setEditingDesign(null);
        }}
        design={editingDesign}
      />

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={!!deleteDesign}
        onClose={() => setDeleteDesign(null)}
        onConfirm={() => deleteDesign && deleteMutation.mutate(deleteDesign.id)}
        title="Delete Design"
        message={`Are you sure you want to delete "${deleteDesign?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
