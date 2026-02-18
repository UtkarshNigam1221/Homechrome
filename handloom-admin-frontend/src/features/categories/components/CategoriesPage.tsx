import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Edit, FolderTree, Plus, Search, Trash2 } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { categoriesApi } from '@/features/categories/api';
import { getErrorMessage } from '@/shared/api/client';
import {
  Badge,
  Button,
  Card,
  ConfirmModal,
  Input,
  Pagination,
} from '@/shared/components/ui';
import { PageLoading } from '@/shared/components/loading';
import { getStatusBadgeVariant } from '@/shared/utils/badge';
import { useCursorPagination } from '@/shared/hooks';
import type { Category } from '../types';

import { CategoryFormModal } from './CategoryFormModal';

export function CategoriesPage() {
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<Category | null>(null);
  const [isFormModalOpen, setIsFormModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const {
    limit,
    cursor,
    hasPrevious,
    goToNextPage,
    goToPreviousPage,
    resetPagination,
    changeLimit,
  } = useCursorPagination(20);

  // Fetch categories list
  const { data, isLoading } = useQuery({
    queryKey: ['categories', { limit, cursor }],
    queryFn: () => categoriesApi.list({ limit, cursor }),
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: categoriesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      toast.success('Category deleted successfully');
      setIsDeleteModalOpen(false);
      setSelectedCategory(null);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const handleEdit = (category: Category) => {
    setSelectedCategory(category);
    setIsFormModalOpen(true);
  };

  const handleDelete = (category: Category) => {
    setSelectedCategory(category);
    setIsDeleteModalOpen(true);
  };

  const handleCreateNew = () => {
    setSelectedCategory(null);
    setIsFormModalOpen(true);
  };

  if (isLoading) {
    return <PageLoading />;
  }

  const categories = data?.items ?? [];
  const pagination = data?.pagination;

  const filteredCategories = searchQuery
    ? categories.filter(
        (cat: Category) =>
          cat.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          cat.slug.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : categories;

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Categories</h1>
          <p className="page-subtitle">Manage your product categories</p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleCreateNew}>
          Add Category
        </Button>
      </div>

      {/* Search */}
      <Card padding="sm">
        <Input
          placeholder="Search categories..."
          value={searchQuery}
          onChange={(e) => {
            setSearchQuery(e.target.value);
            resetPagination();
          }}
          leftIcon={<Search className="w-4 h-4" />}
        />
      </Card>

      {/* Categories Table */}
      <Card padding="none">
        {filteredCategories.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-200 bg-gray-50">
                  <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Category
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Slug
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Products
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Attributes
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="text-right px-6 py-3 text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {filteredCategories.map((category: Category) => (
                  <tr key={category.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-primary-50 rounded-lg flex items-center justify-center flex-shrink-0">
                          <FolderTree className="w-5 h-5 text-primary-600" />
                        </div>
                        <div>
                          <p className="font-medium text-gray-900">{category.name}</p>
                          {category.description && (
                            <p className="text-sm text-gray-500 truncate max-w-xs">
                              {category.description}
                            </p>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span className="text-sm text-gray-500">{category.slug}</span>
                    </td>
                    <td className="px-6 py-4">
                      <span className="text-sm text-gray-700">{category.product_count || 0}</span>
                    </td>
                    <td className="px-6 py-4">
                      <span className="text-sm text-gray-700">
                        {category.own_attributes?.length || 0}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <Badge variant={getStatusBadgeVariant(category.status)}>
                        {category.status}
                      </Badge>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEdit(category)}
                          title="Edit category"
                        >
                          <Edit className="w-4 h-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDelete(category)}
                          title="Delete category"
                          className="text-red-600 hover:text-red-700 hover:bg-red-50"
                        >
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="p-12 text-center">
            <FolderTree className="w-12 h-12 mx-auto text-gray-300 mb-4" />
            <p className="text-gray-500">
              {searchQuery ? 'No categories found matching your search' : 'No categories yet'}
            </p>
            {!searchQuery && (
              <Button
                variant="primary"
                className="mt-4"
                leftIcon={<Plus className="w-4 h-4" />}
                onClick={handleCreateNew}
              >
                Create your first category
              </Button>
            )}
          </div>
        )}

        {/* Pagination */}
        <div className="px-6 border-t border-gray-200">
          <Pagination
            hasMore={pagination?.has_more ?? false}
            hasPrevious={hasPrevious}
            perPage={limit}
            onNextPage={() => pagination?.next_cursor && goToNextPage(pagination.next_cursor)}
            onPreviousPage={goToPreviousPage}
            onPerPageChange={changeLimit}
            itemCount={filteredCategories.length}
          />
        </div>
      </Card>

      {/* Form Modal */}
      <CategoryFormModal
        isOpen={isFormModalOpen}
        onClose={() => {
          setIsFormModalOpen(false);
          setSelectedCategory(null);
        }}
        category={selectedCategory}
      />

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={isDeleteModalOpen}
        onClose={() => {
          setIsDeleteModalOpen(false);
          setSelectedCategory(null);
        }}
        onConfirm={() => selectedCategory && deleteMutation.mutate(selectedCategory.id)}
        title="Delete Category"
        message={`Are you sure you want to delete "${selectedCategory?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
