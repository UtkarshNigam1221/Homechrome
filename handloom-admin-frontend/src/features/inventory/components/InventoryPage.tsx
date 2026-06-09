import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, ArrowDownCircle, ArrowUpCircle, Package, Search } from 'lucide-react';
import { useState } from 'react';

import { inventoryApi } from '@/features/inventory/api';
import { productsApi } from '@/features/products/api';
import type { Product } from '@/features/products/types';
import {
  Badge,
  Button,
  Card,
  Input,
  PageHeader,
  Pagination,
  StatCard,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableLoading,
  TableRow,
} from '@/shared/components/ui';
import { useCursorPagination, useDebounce } from '@/shared/hooks';

import { StockAdjustmentModal } from './StockAdjustmentModal';

export function InventoryPage() {
  const {
    limit,
    cursor,
    hasPrevious,
    goToNextPage,
    goToPreviousPage,
    resetPagination,
    changeLimit,
  } = useCursorPagination(10);
  const [searchQuery, setSearchQuery] = useState('');
  const debouncedSearch = useDebounce(searchQuery, 300);
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);
  const [adjustmentType, setAdjustmentType] = useState<'ADD' | 'REMOVE' | 'ADJUST'>('ADD');
  const [showAdjustmentModal, setShowAdjustmentModal] = useState(false);

  const { data: lowStockData } = useQuery({
    queryKey: ['low-stock', { limit, cursor }],
    queryFn: () => inventoryApi.getLowStock({ limit, cursor }),
  });

  const { data: productsData, isLoading: productsLoading } = useQuery({
    queryKey: ['products-inventory', { limit, cursor, search: debouncedSearch }],
    queryFn: () => productsApi.list({ limit, cursor, search: debouncedSearch || undefined }),
  });

  const lowStockItems = lowStockData?.items ?? [];
  const products = productsData?.items ?? [];
  const pagination = productsData?.pagination;

  const lowStockCount = lowStockItems.length;
  const outOfStockCount = lowStockItems.filter((i) => i.available_qty === 0).length;

  const handleAddStock = (product: Product) => {
    setSelectedProduct(product);
    setAdjustmentType('ADD');
    setShowAdjustmentModal(true);
  };

  const handleRemoveStock = (product: Product) => {
    setSelectedProduct(product);
    setAdjustmentType('REMOVE');
    setShowAdjustmentModal(true);
  };

  return (
    <div className="space-y-6">
      <PageHeader title="Inventory" subtitle="Track and manage product stock levels" />

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <StatCard
          title="Low Stock Items"
          value={lowStockCount}
          icon={<AlertTriangle className="w-6 h-6" />}
          trend={lowStockCount > 0 ? 'down' : 'neutral'}
        />
        <StatCard
          title="Out of Stock"
          value={outOfStockCount}
          icon={<Package className="w-6 h-6" />}
          trend={outOfStockCount > 0 ? 'down' : 'neutral'}
        />
        <StatCard
          title="Total Products"
          value={products.length}
          icon={<Package className="w-6 h-6" />}
        />
      </div>

      {/* Search */}
      <Card padding="sm">
        <Input
          placeholder="Search products by name or SKU..."
          value={searchQuery}
          onChange={(e) => {
            setSearchQuery(e.target.value);
            resetPagination();
          }}
          leftIcon={<Search className="w-4 h-4" />}
        />
      </Card>

      {/* Low Stock Alert */}
      {lowStockCount > 0 && (
        <Card className="border-red-200 bg-red-50" padding="sm">
          <div className="flex items-center gap-3">
            <AlertTriangle className="w-5 h-5 text-red-600" />
            <p className="text-red-800">
              <span className="font-medium">{lowStockCount} products</span> are running low on stock
              and need attention.
            </p>
          </div>
        </Card>
      )}

      {/* Inventory Table */}
      <Card padding="none">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Product</TableHead>
              <TableHead>SKU</TableHead>
              <TableHead>Available</TableHead>
              <TableHead>Reserved</TableHead>
              <TableHead>Threshold</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {productsLoading ? (
              <TableLoading rows={5} colSpan={7} />
            ) : products.length === 0 ? (
              <TableEmpty colSpan={7} message="No products found" />
            ) : (
              products.map((product) => {
                const isLowStock = product.available_qty <= product.low_stock_threshold;
                const isOutOfStock = product.available_qty === 0;

                return (
                  <TableRow key={product.id}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        {product.images?.[0] ? (
                          <img
                            src={product.images[0].url}
                            alt={product.images[0].alt_text || product.name}
                            className="w-10 h-10 rounded-lg object-cover"
                            loading="lazy"
                          />
                        ) : (
                          <div className="w-10 h-10 bg-gray-100 rounded-lg flex items-center justify-center">
                            <Package className="w-5 h-5 text-gray-400" />
                          </div>
                        )}
                        <p className="font-medium">{product.name}</p>
                      </div>
                    </TableCell>
                    <TableCell>
                      <span className="font-mono text-sm">{product.sku}</span>
                    </TableCell>
                    <TableCell>
                      <span className={isLowStock ? 'text-red-600 font-medium' : ''}>
                        {product.available_qty}
                      </span>
                    </TableCell>
                    <TableCell>{product.reserved_qty}</TableCell>
                    <TableCell>{product.low_stock_threshold}</TableCell>
                    <TableCell>
                      {isOutOfStock ? (
                        <Badge variant="danger">Out of Stock</Badge>
                      ) : isLowStock ? (
                        <Badge variant="warning">Low Stock</Badge>
                      ) : (
                        <Badge variant="success">In Stock</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          title="Add stock"
                          className="text-green-600 hover:bg-green-50"
                          onClick={() => handleAddStock(product)}
                        >
                          <ArrowUpCircle className="w-4 h-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          title="Remove stock"
                          className="text-red-600 hover:bg-red-50"
                          onClick={() => handleRemoveStock(product)}
                        >
                          <ArrowDownCircle className="w-4 h-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })
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

      {/* Stock Adjustment Modal */}
      <StockAdjustmentModal
        isOpen={showAdjustmentModal}
        onClose={() => {
          setShowAdjustmentModal(false);
          setSelectedProduct(null);
        }}
        product={selectedProduct}
        adjustmentType={adjustmentType}
      />
    </div>
  );
}
