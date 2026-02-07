import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Edit, Eye, Filter, Package, Plus, Search, Trash2 } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { categoriesApi, getErrorMessage, productsApi } from '../../api';
import {
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
import type { Category, Product } from '../../types';
import { ProductFormModal } from './ProductFormModal';

export function ProductsPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(10);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('');
  const [deleteProduct, setDeleteProduct] = useState<Product | null>(null);
  const [showFilters, setShowFilters] = useState(false);
  const [showFormModal, setShowFormModal] = useState(false);
  const [editingProduct, setEditingProduct] = useState<Product | null>(null);

  // Fetch products
  const { data: productsData, isLoading } = useQuery({
    queryKey: [
      'products',
      {
        page,
        limit: perPage,
        search: searchQuery,
        status: statusFilter,
        category_id: categoryFilter,
      },
    ],
    queryFn: () =>
      productsApi.list({
        page,
        limit: perPage,
        search: searchQuery || undefined,
        status: statusFilter || undefined,
        category_id: categoryFilter || undefined,
      }),
  });

  // Fetch categories for filter
  const { data: categoriesData } = useQuery({
    queryKey: ['categories-list'],
    queryFn: () => categoriesApi.list({ limit: 100 }),
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

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency: 'INR',
      minimumFractionDigits: 0,
    }).format(value / 100);
  };

  // Handle various response formats from the API
  // Backend returns { products: [...], pagination: {...} } or { categories: [...] } etc.
  const extractItems = <T,>(data: unknown, key?: string): T[] => {
    if (!data) return [];
    if (Array.isArray(data)) return data as T[];
    if (typeof data === 'object' && data !== null) {
      const record = data as Record<string, unknown>;
      if (key && key in record) return record[key] as T[];
      if ('items' in record) return record.items as T[];
      if ('data' in record) return Array.isArray(record.data) ? (record.data as T[]) : [];
    }
    return [];
  };

  const products = extractItems<Product>(productsData, 'products');
  const pagination = productsData?.pagination;

  const categories = extractItems<Category>(categoriesData, 'categories');
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
                setPage(1);
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
          <div className="mt-4 pt-4 border-t border-gray-200 grid grid-cols-1 md:grid-cols-3 gap-4">
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
                setPage(1);
              }}
            />
            <Select
              label="Category"
              options={categoryOptions}
              value={categoryFilter}
              onChange={(e) => {
                setCategoryFilter(e.target.value);
                setPage(1);
              }}
            />
            <div className="flex items-end">
              <Button
                variant="ghost"
                onClick={() => {
                  setStatusFilter('');
                  setCategoryFilter('');
                  setSearchQuery('');
                  setPage(1);
                }}
              >
                Clear Filters
              </Button>
            </div>
          </div>
        )}
      </Card>

      {/* Products Table */}
      <Card padding="none">
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

        {pagination && pagination.total_pages > 1 && (
          <div className="border-t border-gray-200 px-6">
            <Pagination
              currentPage={pagination.current_page}
              totalPages={pagination.total_pages}
              totalCount={pagination.total_count}
              perPage={pagination.per_page}
              onPageChange={setPage}
              onPerPageChange={(newPerPage) => {
                setPerPage(newPerPage);
                setPage(1);
              }}
            />
          </div>
        )}
      </Card>

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
