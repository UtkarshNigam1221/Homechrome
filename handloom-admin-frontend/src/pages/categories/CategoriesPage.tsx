import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { clsx } from 'clsx';
import { ChevronRight, Edit, FolderTree, Plus, Search, Trash2 } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { categoriesApi, getErrorMessage } from '../../api';
import {
  Badge,
  Button,
  Card,
  ConfirmModal,
  getStatusBadgeVariant,
  Input,
  PageLoading,
} from '../../components/common';
import type { Category } from '../../types';
import { CategoryFormModal } from './CategoryFormModal';

export function CategoriesPage() {
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<Category | null>(null);
  const [isFormModalOpen, setIsFormModalOpen] = useState(false);
  const [isDeleteModalOpen, setIsDeleteModalOpen] = useState(false);
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set());

  // Fetch categories tree
  const { data: categories, isLoading } = useQuery({
    queryKey: ['categories-tree'],
    queryFn: categoriesApi.getTree,
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: categoriesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories-tree'] });
      toast.success('Category deleted successfully');
      setIsDeleteModalOpen(false);
      setSelectedCategory(null);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const toggleExpand = (categoryId: string) => {
    const newExpanded = new Set(expandedCategories);
    if (newExpanded.has(categoryId)) {
      newExpanded.delete(categoryId);
    } else {
      newExpanded.add(categoryId);
    }
    setExpandedCategories(newExpanded);
  };

  const handleEdit = (category: Category) => {
    setSelectedCategory(category);
    setIsFormModalOpen(true);
  };

  const handleDelete = (category: Category) => {
    setSelectedCategory(category);
    setIsDeleteModalOpen(true);
  };

  const handleCreateNew = (parentId?: string) => {
    if (parentId) {
      setSelectedCategory({ parent_id: parentId } as Category);
    } else {
      setSelectedCategory(null);
    }
    setIsFormModalOpen(true);
  };

  const filterCategories = (cats: Category[], query: string): Category[] => {
    if (!query) return cats;

    return cats.filter((cat) => {
      const matches =
        cat.name.toLowerCase().includes(query.toLowerCase()) ||
        cat.slug.toLowerCase().includes(query.toLowerCase());

      if (matches) return true;

      if (cat.children && cat.children.length > 0) {
        const filteredChildren = filterCategories(cat.children, query);
        if (filteredChildren.length > 0) {
          cat.children = filteredChildren;
          return true;
        }
      }

      return false;
    });
  };

  const renderCategoryItem = (category: Category, depth = 0) => {
    const hasChildren = category.children && category.children.length > 0;
    const isExpanded = expandedCategories.has(category.id);

    return (
      <div key={category.id}>
        <div
          className={clsx(
            'flex items-center justify-between p-4 hover:bg-gray-50 transition-colors border-b border-gray-100',
            depth > 0 && 'bg-gray-50/50'
          )}
          style={{ paddingLeft: `${16 + depth * 24}px` }}
        >
          <div className="flex items-center gap-3">
            {hasChildren ? (
              <button
                onClick={() => toggleExpand(category.id)}
                className="p-1 rounded hover:bg-gray-200 transition-colors"
              >
                <ChevronRight
                  className={clsx(
                    'w-4 h-4 text-gray-400 transition-transform',
                    isExpanded && 'transform rotate-90'
                  )}
                />
              </button>
            ) : (
              <div className="w-6" />
            )}
            <div className="w-10 h-10 bg-primary-50 rounded-lg flex items-center justify-center">
              <FolderTree className="w-5 h-5 text-primary-600" />
            </div>
            <div>
              <p className="font-medium text-gray-900">{category.name}</p>
              <div className="flex items-center gap-2 text-sm text-gray-500">
                <span>{category.slug}</span>
                <span>•</span>
                <span>{category.product_count || 0} products</span>
                {category.allow_custom_dimensions && (
                  <>
                    <span>•</span>
                    <span className="text-primary-600">Custom dimensions</span>
                  </>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <Badge variant={getStatusBadgeVariant(category.status)}>{category.status}</Badge>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => handleCreateNew(category.id)}
                title="Add subcategory"
              >
                <Plus className="w-4 h-4" />
              </Button>
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
          </div>
        </div>
        {hasChildren && isExpanded && category.children && (
          <div>{category.children.map((child) => renderCategoryItem(child, depth + 1))}</div>
        )}
      </div>
    );
  };

  if (isLoading) {
    return <PageLoading />;
  }

  const filteredCategories = searchQuery
    ? filterCategories(JSON.parse(JSON.stringify(categories || [])), searchQuery)
    : categories || [];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Categories</h1>
          <p className="page-subtitle">Organize your products with hierarchical categories</p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={() => handleCreateNew()}>
          Add Category
        </Button>
      </div>

      {/* Search */}
      <Card padding="sm">
        <Input
          placeholder="Search categories..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          leftIcon={<Search className="w-4 h-4" />}
        />
      </Card>

      {/* Categories Tree */}
      <Card padding="none">
        {filteredCategories.length > 0 ? (
          filteredCategories.map((category) => renderCategoryItem(category))
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
                onClick={() => handleCreateNew()}
              >
                Create your first category
              </Button>
            )}
          </div>
        )}
      </Card>

      {/* Form Modal */}
      <CategoryFormModal
        isOpen={isFormModalOpen}
        onClose={() => {
          setIsFormModalOpen(false);
          setSelectedCategory(null);
        }}
        category={selectedCategory}
        categories={categories || []}
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
