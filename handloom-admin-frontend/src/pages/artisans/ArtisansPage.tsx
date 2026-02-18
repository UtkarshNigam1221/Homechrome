import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Edit, MapPin, Phone, Plus, Search, Trash2, User } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { artisansApi, getErrorMessage } from '../../api';
import {
  Badge,
  Button,
  Card,
  ConfirmModal,
  getStatusBadgeVariant,
  Input,
  Pagination,
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
import type { Artisan } from '../../types';
import { formatCurrency } from '@/utils/currency';
import { ArtisanFormModal } from './ArtisanFormModal';

export function ArtisansPage() {
  const queryClient = useQueryClient();
  const { limit, cursor, hasPrevious, goToNextPage, goToPreviousPage, resetPagination, changeLimit } = useCursorPagination(10);
  const [searchQuery, setSearchQuery] = useState('');
  const [showFormModal, setShowFormModal] = useState(false);
  const [editingArtisan, setEditingArtisan] = useState<Artisan | null>(null);
  const [deleteArtisan, setDeleteArtisan] = useState<Artisan | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['artisans', { limit, cursor, search: searchQuery }],
    queryFn: () => artisansApi.list({ limit, cursor, search: searchQuery || undefined }),
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: artisansApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['artisans'] });
      toast.success('Artisan deleted successfully');
      setDeleteArtisan(null);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const artisans = data?.items || [];
  const pagination = data?.pagination;

  const handleOpenCreate = () => {
    setEditingArtisan(null);
    setShowFormModal(true);
  };

  const handleEdit = (artisan: Artisan) => {
    setEditingArtisan(artisan);
    setShowFormModal(true);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Artisans</h1>
          <p className="page-subtitle">Manage your artisan network</p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
          Add Artisan
        </Button>
      </div>

      <Card padding="sm">
        <Input
          placeholder="Search artisans..."
          value={searchQuery}
          onChange={(e) => {
            setSearchQuery(e.target.value);
            resetPagination();
          }}
          leftIcon={<Search className="w-4 h-4" />}
        />
      </Card>

      <Card padding="none">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Artisan</TableHead>
              <TableHead>Contact</TableHead>
              <TableHead>Location</TableHead>
              <TableHead>Skills</TableHead>
              <TableHead>Products</TableHead>
              <TableHead>Earnings</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={5} colSpan={8} />
            ) : artisans.length === 0 ? (
              <TableEmpty
                colSpan={8}
                message="No artisans found"
                action={
                  <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
                    Add your first artisan
                  </Button>
                }
              />
            ) : (
              artisans.map((artisan) => (
                <TableRow key={artisan.id} clickable onClick={() => handleEdit(artisan)}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-primary-100 rounded-full flex items-center justify-center">
                        <User className="w-5 h-5 text-primary-600" />
                      </div>
                      <div>
                        <p className="font-medium">{artisan.name}</p>
                        <p className="text-sm text-gray-500">{artisan.craft_type}</p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1 text-sm text-gray-500">
                      <Phone className="w-3 h-3" />
                      {artisan.phone}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1 text-sm text-gray-500">
                      <MapPin className="w-3 h-3" />
                      {artisan.location?.city}, {artisan.location?.state}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1 flex-wrap">
                      {artisan.skills?.slice(0, 2).map((skill) => (
                        <Badge key={skill} variant="gray" size="sm">
                          {skill}
                        </Badge>
                      ))}
                      {artisan.skills && artisan.skills.length > 2 && (
                        <Badge variant="gray" size="sm">
                          +{artisan.skills.length - 2}
                        </Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>{artisan.product_count || 0}</TableCell>
                  <TableCell>{formatCurrency(artisan.total_earnings || 0)}</TableCell>
                  <TableCell>
                    <Badge variant={getStatusBadgeVariant(artisan.status)}>{artisan.status}</Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleEdit(artisan);
                        }}
                        title="Edit artisan"
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteArtisan(artisan);
                        }}
                        title="Delete artisan"
                        className="text-red-600 hover:text-red-700 hover:bg-red-50"
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
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
              itemCount={artisans.length}
            />
          </div>
        )}
      </Card>

      {/* Artisan Form Modal */}
      <ArtisanFormModal
        isOpen={showFormModal}
        onClose={() => {
          setShowFormModal(false);
          setEditingArtisan(null);
        }}
        artisan={editingArtisan}
      />

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={!!deleteArtisan}
        onClose={() => setDeleteArtisan(null)}
        onConfirm={() => deleteArtisan && deleteMutation.mutate(deleteArtisan.id)}
        title="Delete Artisan"
        message={`Are you sure you want to delete "${deleteArtisan?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
