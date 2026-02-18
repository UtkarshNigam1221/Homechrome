import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { DollarSign, Edit, Plus, Search, Trash2 } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { getErrorMessage, pricingApi } from '@/api';
import {
  Badge,
  Button,
  Card,
  ConfirmModal,
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
} from '@/components/common';
import { useCursorPagination } from '@/hooks';
import type { PricingRule } from '@/types';
import { formatCurrency } from '@/utils/currency';
import { PricingRuleFormModal } from './PricingRuleFormModal';

export function PricingRulesPage() {
  const queryClient = useQueryClient();
  const { limit, cursor, hasPrevious, goToNextPage, goToPreviousPage, resetPagination, changeLimit } = useCursorPagination(10);
  const [searchQuery, setSearchQuery] = useState('');
  const [showFormModal, setShowFormModal] = useState(false);
  const [editingRule, setEditingRule] = useState<PricingRule | null>(null);
  const [deleteRule, setDeleteRule] = useState<PricingRule | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['pricing-rules', { limit, cursor }],
    queryFn: () => pricingApi.listRules({ limit, cursor }),
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: pricingApi.deleteRule,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pricing-rules'] });
      toast.success('Pricing rule deleted successfully');
      setDeleteRule(null);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const rules = data?.items || [];
  const pagination = data?.pagination;

  const handleOpenCreate = () => {
    setEditingRule(null);
    setShowFormModal(true);
  };

  const handleEdit = (rule: PricingRule) => {
    setEditingRule(rule);
    setShowFormModal(true);
  };

  // Filter by search (client-side for now)
  const filteredRules = searchQuery
    ? rules.filter((r) => r.name.toLowerCase().includes(searchQuery.toLowerCase()))
    : rules;

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Pricing Rules</h1>
          <p className="page-subtitle">Configure dynamic pricing for products</p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
          Add Rule
        </Button>
      </div>

      <Card padding="sm">
        <Input
          placeholder="Search rules..."
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
              <TableHead>Rule Name</TableHead>
              <TableHead>Scope</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Base Price</TableHead>
              <TableHead>Priority</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={5} colSpan={7} />
            ) : filteredRules.length === 0 ? (
              <TableEmpty
                colSpan={7}
                message="No pricing rules found"
                action={
                  <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
                    Create your first rule
                  </Button>
                }
              />
            ) : (
              filteredRules.map((rule) => (
                <TableRow key={rule.id} clickable onClick={() => handleEdit(rule)}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-green-50 rounded-lg flex items-center justify-center">
                        <DollarSign className="w-5 h-5 text-green-600" />
                      </div>
                      <div>
                        <p className="font-medium">{rule.name}</p>
                        <p className="text-sm text-gray-500 truncate max-w-xs">
                          {rule.description}
                        </p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="info">{rule.scope_type}</Badge>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm">{rule.pricing_type.replace('_', ' ')}</span>
                  </TableCell>
                  <TableCell>{formatCurrency(rule.base_price)}</TableCell>
                  <TableCell>{rule.priority}</TableCell>
                  <TableCell>
                    <Badge variant={rule.is_active ? 'success' : 'gray'}>
                      {rule.is_active ? 'Active' : 'Inactive'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleEdit(rule);
                        }}
                        title="Edit rule"
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteRule(rule);
                        }}
                        title="Delete rule"
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
              itemCount={filteredRules.length}
            />
          </div>
        )}
      </Card>

      {/* Pricing Rule Form Modal */}
      <PricingRuleFormModal
        isOpen={showFormModal}
        onClose={() => {
          setShowFormModal(false);
          setEditingRule(null);
        }}
        rule={editingRule}
      />

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={!!deleteRule}
        onClose={() => setDeleteRule(null)}
        onConfirm={() => deleteRule && deleteMutation.mutate(deleteRule.id)}
        title="Delete Pricing Rule"
        message={`Are you sure you want to delete "${deleteRule?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
