import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Edit, Eye, Filter, Package, Plus, Search, Trash2, X } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { categoriesApi, getErrorMessage, productsApi } from '../../api';
import {
  AttributeFilterSidebar,
  Badge,
  Button,
  Card,
  ConfirmModal,
  getStatusBadgeVariant,
  Input,
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
import type { CategoryAttribute, Product } from '../../types';
import { formatCurrency } from '@/utils/currency';
import { ProductFormModal } from './ProductFormModal';

export function ProductsPage() {
  const queryClient = useQueryClient();
  const { limit, cursor, hasPrevious, goToNextPage, goToPreviousPage, resetPagination, changeLimit } = useCursorPagination(10);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('');
  const [attributeFilters, setAttributeFilters] = useState<Record<string, string[]>>({});
  const [deleteProduct, setDeleteProduct] = useState<Product | null>(null);
  const [showFilters, setShowFilters] = useState(false);
  const [showAttributeFilters, setShowAttributeFilters] = useState(false);
  const [showFormModal, setShowFormModal] = useState(false);
  const [editingProduct, setEditingProduct] = useState<Product | null>(null);

  // Fetch products
  const { data: productsData, isLoading } = useQuery({
    queryKey: [
      'products',
      {
        limit,
        cursor,
        search: searchQuery,
        status: statusFilter,
        category_id: categoryFilter,
        attribute_filters: attributeFilters,
      },
    ],
    queryFn: () =>
      productsApi.list({
        limit,
        cursor,
        search: searchQuery || undefined,
        status: statusFilter || undefined,
        category_id: categoryFilter || undefined,
        attribute_filters: Object.keys(attributeFilters).length > 0 ? attributeFilters : undefined,
      }),
  });

  // Fetch categories for filter
  const { data: categoriesData } = useQuery({
    queryKey: ['categories-list'],
    queryFn: () => categoriesApi.list({ limit: 100 }),
  });

  // Fetch category attributes when a category is selected
  const { data: categoryAttributesData } = useQuery({
    queryKey: ['category-attributes', categoryFilter],
    queryFn: () => categoriesApi.getAttributes(categoryFilter),
    enabled: !!categoryFilter, // Only fetch when category is selected
  });

  // Fetch distinct attribute filter options from GSI (optimized — no product data loaded)
  const { data: filterOptionsData } = useQuery({
    queryKey: ['product-filter-options', categoryFilter],
    queryFn: () => productsApi.getFilterOptions(categoryFilter),
    enabled: !!categoryFilter,
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: productsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      toast.success('Product deleted successfully');
      setDeleteProduct(null);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const products = productsData?.items ?? [];
  const pagination = productsData?.pagination;

  // Build category attributes enriched with distinct values from GSI
  const categoryAttributes: CategoryAttribute[] = (() => {
    if (!categoryFilter) return [];
    const rawAttrs = categoryAttributesData?.own_attributes || [];
    const filterOptions: Record<string, string[]> = filterOptionsData || {};

    return rawAttrs.map((attr: CategoryAttribute) => {
      // If attribute already has options (SELECT/MULTI_SELECT), use them as-is
      if (attr.options && attr.options.length > 0) return attr;

      // For TEXT/NUMBER attributes without predefined options, use GSI-discovered values
      if (!attr.searchable) return attr;

      const distinctValues = filterOptions[attr.name];
      if (!distinctValues || distinctValues.length === 0) return attr;

      // Build options from GSI distinct values
      const discoveredOptions = distinctValues.map((v) => ({ value: v, label: v }));
      return { ...attr, options: discoveredOptions };
    });
  })();

  const categories = categoriesData?.items ?? [];
  const categoryOptions = [
    { value: '', label: 'All Categories' },
    ...categories.map((cat) => ({ value: cat.id, label: cat.name })),
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Products</h1>
          <p className="page-subtitle">Manage your product catalog</p>
        </div>
        <Button
          leftIcon={<Plus className="w-4 h-4" />}
          onClick={() => {
            setEditingProduct(null);
            setShowFormModal(true);
          }}
        >
          Add Product
        </Button>
      </div>

      {/* Filters */}
      <Card padding="sm">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1">
            <Input
              placeholder="Search products by name, SKU..."
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                resetPagination();
              }}
              leftIcon={<Search className="w-4 h-4" />}
            />
          </div>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              leftIcon={<Filter className="w-4 h-4" />}
              onClick={() => setShowFilters(!showFilters)}
            >
              Filters
            </Button>
          </div>
        </div>

        {showFilters && (
          <div className="mt-4 pt-4 border-t border-gray-200 grid grid-cols-1 md:grid-cols-4 gap-4">
            <Select
              label="Status"
              options={[
                { value: '', label: 'All Statuses' },
                { value: 'ACTIVE', label: 'Active' },
                { value: 'INACTIVE', label: 'Inactive' },
                { value: 'DRAFT', label: 'Draft' },
              ]}
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                resetPagination();
              }}
            />
            <Select
              label="Category"
              options={categoryOptions}
              value={categoryFilter}
              onChange={(e) => {
                setCategoryFilter(e.target.value);
                setAttributeFilters({}); // Clear attribute filters when category changes
                resetPagination();
              }}
            />
            {categoryFilter && categoryAttributes.some((a) => a.searchable && a.options?.length) && (
              <div className="flex items-end">
                <Button
                  variant={showAttributeFilters ? 'primary' : 'secondary'}
                  onClick={() => setShowAttributeFilters(!showAttributeFilters)}
                >
                  <Filter className="w-4 h-4 mr-2" />
                  Attribute Filters
                  {Object.keys(attributeFilters).length > 0 && (
                    <span className="ml-2 inline-flex items-center justify-center px-2 py-0.5 text-xs font-medium bg-white text-indigo-600 rounded-full">
                      {Object.values(attributeFilters).reduce((sum, arr) => sum + arr.length, 0)}
                    </span>
                  )}
                </Button>
              </div>
            )}
            <div className="flex items-end">
              <Button
                variant="ghost"
                onClick={() => {
                  setStatusFilter('');
                  setCategoryFilter('');
                  setSearchQuery('');
                  setAttributeFilters({});
                  setShowAttributeFilters(false);
                  resetPagination();
                }}
              >
                Clear All
              </Button>
            </div>
          </div>
        )}

        {/* Active Attribute Filters Display */}
        {Object.keys(attributeFilters).length > 0 && (
          <div className="mt-4 pt-4 border-t border-gray-200">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm text-gray-500">Active filters:</span>
              {Object.entries(attributeFilters).map(([attrName, values]) => {
                const attr = categoryAttributes.find((a) => a.name === attrName);
                return values.map((value) => {
                  const option = attr?.options?.find((o) => o.value === value);
                  return (
                    <span
                      key={`${attrName}-${value}`}
                      className="inline-flex items-center gap-1 px-2 py-1 text-sm bg-indigo-100 text-indigo-800 rounded-full"
                    >
                      <span className="font-medium">{attr?.label || attrName}:</span>
                      {option?.label || value}
                      <button
                        type="button"
                        onClick={() => {
                          const newValues = attributeFilters[attrName].filter((v) => v !== value);
                          const newFilters = { ...attributeFilters };
                          if (newValues.length > 0) {
                            newFilters[attrName] = newValues;
                          } else {
                            delete newFilters[attrName];
                          }
                          setAttributeFilters(newFilters);
                          resetPagination();
                        }}
                        className="ml-1 hover:text-indigo-600"
                      >
                        <X className="w-3 h-3" />
                      </button>
                    </span>
                  );
                });
              })}
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setAttributeFilters({});
                  resetPagination();
                }}
                className="text-gray-500"
              >
                Clear all filters
              </Button>
            </div>
          </div>
        )}
      </Card>

      {/* Products Table with Optional Filter Sidebar */}
      <div className={`flex gap-6 ${showAttributeFilters ? '' : ''}`}>
        {/* Attribute Filter Sidebar */}
        {showAttributeFilters && categoryFilter && (
          <div className="w-64 flex-shrink-0">
            <AttributeFilterSidebar
              attributes={categoryAttributes}
              selectedFilters={attributeFilters}
              onFilterChange={(filters) => {
                setAttributeFilters(filters);
                resetPagination();
              }}
              onClearFilters={() => {
                setAttributeFilters({});
                resetPagination();
              }}
              isLoading={isLoading}
            />
          </div>
        )}

        {/* Products Table */}
        <Card padding="none" className="flex-1">
          <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Product</TableHead>
              <TableHead>SKU</TableHead>
              <TableHead>Category</TableHead>
              <TableHead>Price</TableHead>
              <TableHead>Stock</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={5} colSpan={7} />
            ) : products.length === 0 ? (
              <TableEmpty
                colSpan={7}
                message="No products found"
                description={
                  searchQuery || statusFilter || categoryFilter
                    ? 'Try adjusting your filters'
                    : 'Start by adding your first product'
                }
                action={
                  !searchQuery &&
                  !statusFilter &&
                  !categoryFilter && (
                    <Button
                      leftIcon={<Plus className="w-4 h-4" />}
                      onClick={() => {
                        setEditingProduct(null);
                        setShowFormModal(true);
                      }}
                    >
                      Add Product
                    </Button>
                  )
                }
              />
            ) : (
              products.map((product) => (
                <TableRow
                  key={product.id}
                  clickable
                  onClick={() => {
                    setEditingProduct(product);
                    setShowFormModal(true);
                  }}
                >
                  <TableCell>
                    <div className="flex items-center gap-3">
                      {product.images?.[0] ? (
                        <img
                          src={product.images[0].url}
                          alt={product.images[0].alt_text || product.name}
                          className="w-10 h-10 rounded-lg object-cover"
                        />
                      ) : (
                        <div className="w-10 h-10 bg-gray-100 rounded-lg flex items-center justify-center">
                          <Package className="w-5 h-5 text-gray-400" />
                        </div>
                      )}
                      <div>
                        <p className="font-medium text-gray-900">{product.name}</p>
                        {product.material && (
                          <p className="text-sm text-gray-500">{product.material}</p>
                        )}
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-sm">{product.sku}</span>
                  </TableCell>
                  <TableCell>
                    <span className="text-gray-600">{product.category_id}</span>
                  </TableCell>
                  <TableCell>
                    <div>
                      <p className="font-medium">{formatCurrency(product.selling_price)}</p>
                      {product.base_price !== product.selling_price && (
                        <p className="text-sm text-gray-500 line-through">
                          {formatCurrency(product.base_price)}
                        </p>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span
                        className={
                          product.available_qty <= product.low_stock_threshold
                            ? 'text-red-600 font-medium'
                            : ''
                        }
                      >
                        {product.available_qty}
                      </span>
                      {product.available_qty <= product.low_stock_threshold && (
                        <Badge variant="danger" size="sm">
                          Low
                        </Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getStatusBadgeVariant(product.status)}>{product.status}</Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setEditingProduct(product);
                          setShowFormModal(true);
                        }}
                        title="View product"
                      >
                        <Eye className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setEditingProduct(product);
                          setShowFormModal(true);
                        }}
                        title="Edit product"
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteProduct(product);
                        }}
                        title="Delete product"
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
              itemCount={products.length}
            />
          </div>
        )}
        </Card>
      </div>

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={!!deleteProduct}
        onClose={() => setDeleteProduct(null)}
        onConfirm={() => deleteProduct && deleteMutation.mutate(deleteProduct.id)}
        title="Delete Product"
        message={`Are you sure you want to delete "${deleteProduct?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        loading={deleteMutation.isPending}
      />

      {/* Product Form Modal */}
      <ProductFormModal
        isOpen={showFormModal}
        onClose={() => {
          setShowFormModal(false);
          setEditingProduct(null);
        }}
        product={editingProduct}
      />
    </div>
  );
}
