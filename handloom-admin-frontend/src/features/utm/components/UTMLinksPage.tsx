import { useQuery } from '@tanstack/react-query';
import { Copy, Edit, Link2, Plus, Search, Trash2 } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { utmLinksApi } from '@/features/utm/api';
import {
  Badge,
  Button,
  Card,
  Input,
  PageHeader,
  Pagination,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableLoading,
  TableRow,
} from '@/shared/components/ui';
import { useCursorPagination, useDebounce, useDeleteWithConfirm } from '@/shared/hooks';

import type { UTMLink } from '../types';
import { UTMLinkFormModal } from './UTMLinkFormModal';

const destLabel: Record<UTMLink['dest_type'], string> = {
  HOME: 'Home',
  CATEGORY: 'Category',
  PRODUCT: 'Product',
};

export function UTMLinksPage() {
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
  const [showFormModal, setShowFormModal] = useState(false);
  const [editingLink, setEditingLink] = useState<UTMLink | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['utm-links', { limit, cursor, search: debouncedSearch }],
    queryFn: () => utmLinksApi.list({ limit, cursor, search: debouncedSearch || undefined }),
  });

  const { setDeleteTarget: setDeleteLink, DeleteConfirmModal } = useDeleteWithConfirm<UTMLink>({
    queryKey: 'utm-links',
    deleteFn: utmLinksApi.delete,
    entityName: 'UTM link',
    getEntityName: (l) => l.name,
  });

  const links = data?.items || [];
  const pagination = data?.pagination;

  const handleOpenCreate = () => {
    setEditingLink(null);
    setShowFormModal(true);
  };

  const handleEdit = (link: UTMLink) => {
    setEditingLink(link);
    setShowFormModal(true);
  };

  const handleCopy = async (url: string) => {
    await navigator.clipboard.writeText(url);
    toast.success('Link copied');
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="UTM Links"
        subtitle="Build and store tagged campaign links for the live store"
        action={
          <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
            Create Link
          </Button>
        }
      />

      <Card padding="sm">
        <Input
          placeholder="Search by name or campaign..."
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
              <TableHead>Name</TableHead>
              <TableHead>Destination</TableHead>
              <TableHead>Source / Medium / Campaign</TableHead>
              <TableHead>URL</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={5} colSpan={5} />
            ) : links.length === 0 ? (
              <TableEmpty
                colSpan={5}
                message="No UTM links found"
                action={
                  <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
                    Create your first link
                  </Button>
                }
              />
            ) : (
              links.map((link) => (
                <TableRow key={link.id} clickable onClick={() => handleEdit(link)}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-blue-50 rounded-lg flex items-center justify-center">
                        <Link2 className="w-5 h-5 text-blue-600" />
                      </div>
                      <p className="font-medium">{link.name}</p>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="gray" size="sm">
                      {destLabel[link.dest_type]}
                    </Badge>
                    {link.dest_slug && (
                      <p className="font-mono text-xs text-gray-500 mt-1">{link.dest_slug}</p>
                    )}
                  </TableCell>
                  <TableCell>
                    <p className="font-mono text-xs">
                      {link.utm_source} / {link.utm_medium} / {link.utm_campaign}
                    </p>
                  </TableCell>
                  <TableCell>
                    <p className="font-mono text-xs break-all max-w-xs text-gray-600">{link.url}</p>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          void handleCopy(link.url);
                        }}
                        title="Copy link"
                      >
                        <Copy className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleEdit(link);
                        }}
                        title="Edit link"
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteLink(link);
                        }}
                        title="Delete link"
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
        <div className="border-t border-gray-200 px-6">
          <Pagination
            hasMore={pagination?.has_more ?? false}
            hasPrevious={hasPrevious}
            perPage={limit}
            onNextPage={() => pagination?.next_cursor && goToNextPage(pagination.next_cursor)}
            onPreviousPage={goToPreviousPage}
            onPerPageChange={changeLimit}
            itemCount={links.length}
          />
        </div>
      </Card>

      <UTMLinkFormModal
        isOpen={showFormModal}
        onClose={() => {
          setShowFormModal(false);
          setEditingLink(null);
        }}
        link={editingLink}
      />

      <DeleteConfirmModal />
    </div>
  );
}
