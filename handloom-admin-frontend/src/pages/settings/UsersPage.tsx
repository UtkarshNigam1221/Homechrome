import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { Edit, Plus, Search, Shield, Trash2, User, UserCheck, UserX } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { getErrorMessage, usersApi } from '../../api';
import {
  Badge,
  Button,
  Card,
  ConfirmModal,
  getStatusBadgeVariant,
  Input,
  Pagination,
  Select,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableRow,
} from '../../components/common';
import { useDebounce } from '../../hooks';
import type { User as UserType, UserRole, UserStatus } from '../../types';
import { UserFormModal } from './UserFormModal';

// Skeleton for Users table rows
function UsersTableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <tr key={i} className="border-b border-gray-100">
          {/* User column */}
          <td className="px-6 py-4">
            <div className="flex items-center gap-3">
              <Skeleton variant="circular" width={40} height={40} />
              <div>
                <Skeleton className="h-4 w-32 mb-2" />
                <Skeleton className="h-3 w-40" />
              </div>
            </div>
          </td>
          {/* Role */}
          <td className="px-6 py-4">
            <Skeleton variant="rectangular" className="h-6 w-20 rounded-full" />
          </td>
          {/* Status */}
          <td className="px-6 py-4">
            <Skeleton variant="rectangular" className="h-6 w-16 rounded-full" />
          </td>
          {/* Phone */}
          <td className="px-6 py-4">
            <Skeleton className="h-4 w-24" />
          </td>
          {/* Last Login */}
          <td className="px-6 py-4">
            <Skeleton className="h-4 w-32" />
          </td>
          {/* Created */}
          <td className="px-6 py-4">
            <Skeleton className="h-4 w-24" />
          </td>
          {/* Actions */}
          <td className="px-6 py-4">
            <div className="flex items-center justify-end gap-1">
              <Skeleton variant="rectangular" width={32} height={32} className="rounded" />
              <Skeleton variant="rectangular" width={32} height={32} className="rounded" />
              <Skeleton variant="rectangular" width={32} height={32} className="rounded" />
            </div>
          </td>
        </tr>
      ))}
    </>
  );
}

export function UsersPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(10);
  const [searchInput, setSearchInput] = useState('');
  const [roleFilter, setRoleFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [deleteUser, setDeleteUser] = useState<UserType | null>(null);
  const [showFormModal, setShowFormModal] = useState(false);
  const [editingUser, setEditingUser] = useState<UserType | null>(null);
  const [statusChangeUser, setStatusChangeUser] = useState<{
    user: UserType;
    newStatus: UserStatus;
  } | null>(null);

  // Debounce search query with 2 second delay
  const debouncedSearch = useDebounce(searchInput, 2000);

  const { data, isLoading } = useQuery({
    queryKey: [
      'users',
      { page, limit: perPage, search: debouncedSearch, role: roleFilter, status: statusFilter },
    ],
    queryFn: () =>
      usersApi.list({
        page,
        limit: perPage,
        search: debouncedSearch || undefined,
        role: roleFilter || undefined,
        status: statusFilter || undefined,
      }),
  });

  const deleteMutation = useMutation({
    mutationFn: usersApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast.success('User deleted successfully');
      setDeleteUser(null);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const updateStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      usersApi.updateStatus(id, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast.success('User status updated successfully');
      setStatusChangeUser(null);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const users = data?.items || [];
  const pagination = data?.pagination;

  const getRoleBadgeVariant = (role: UserRole) => {
    switch (role) {
      case 'ADMIN':
        return 'danger';
      case 'OPERATOR':
        return 'warning';
      default:
        return 'info';
    }
  };

  const handleOpenCreate = () => {
    setEditingUser(null);
    setShowFormModal(true);
  };

  const handleEdit = (user: UserType) => {
    setEditingUser(user);
    setShowFormModal(true);
  };

  const handleToggleStatus = (user: UserType, e: React.MouseEvent) => {
    e.stopPropagation();
    const newStatus: UserStatus = user.status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE';
    setStatusChangeUser({ user, newStatus });
  };

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSearchInput(e.target.value);
    setPage(1);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Users</h1>
          <p className="page-subtitle">Manage admin users and permissions</p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
          Add User
        </Button>
      </div>

      {/* Filters */}
      <Card padding="sm">
        <div className="flex flex-col lg:flex-row gap-4">
          <div className="flex-1 min-w-0">
            <Input
              placeholder="Search users by name or email..."
              value={searchInput}
              onChange={handleSearchChange}
              leftIcon={<Search className="w-4 h-4" />}
              className="w-full"
            />
          </div>
          <div className="flex flex-col sm:flex-row gap-4 sm:flex-shrink-0">
            <Select
              options={[
                { value: '', label: 'All Roles' },
                { value: 'ADMIN', label: 'Admin' },
                { value: 'OPERATOR', label: 'Operator' },
              ]}
              value={roleFilter}
              onChange={(e) => {
                setRoleFilter(e.target.value);
                setPage(1);
              }}
              className="sm:w-36"
            />
            <Select
              options={[
                { value: '', label: 'All Status' },
                { value: 'ACTIVE', label: 'Active' },
                { value: 'INACTIVE', label: 'Inactive' },
                { value: 'PENDING', label: 'Pending' },
              ]}
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setPage(1);
              }}
              className="sm:w-36"
            />
          </div>
        </div>
      </Card>

      {/* Users Table */}
      <Card padding="none">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>User</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Phone</TableHead>
              <TableHead>Last Login</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <UsersTableSkeleton rows={5} />
            ) : users.length === 0 ? (
              <TableEmpty
                colSpan={7}
                message="No users found"
                action={
                  <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
                    Create your first user
                  </Button>
                }
              />
            ) : (
              users.map((user) => (
                <TableRow key={user.id} clickable onClick={() => handleEdit(user)}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-primary-100 rounded-full flex items-center justify-center">
                        {user.role === 'ADMIN' ? (
                          <Shield className="w-5 h-5 text-primary-600" />
                        ) : (
                          <User className="w-5 h-5 text-primary-600" />
                        )}
                      </div>
                      <div>
                        <p className="font-medium">
                          {user.first_name} {user.last_name || ''}
                        </p>
                        <p className="text-sm text-gray-500">{user.email}</p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getRoleBadgeVariant(user.role)}>{user.role}</Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getStatusBadgeVariant(user.status)}>{user.status}</Badge>
                  </TableCell>
                  <TableCell>{user.phone || <span className="text-gray-400">-</span>}</TableCell>
                  <TableCell>
                    {user.last_login_at ? (
                      format(new Date(user.last_login_at), 'MMM d, yyyy HH:mm')
                    ) : (
                      <span className="text-gray-400">Never</span>
                    )}
                  </TableCell>
                  <TableCell>{format(new Date(user.created_at), 'MMM d, yyyy')}</TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        title={user.status === 'ACTIVE' ? 'Deactivate user' : 'Activate user'}
                        onClick={(e) => handleToggleStatus(user, e)}
                      >
                        {user.status === 'ACTIVE' ? (
                          <UserX className="w-4 h-4 text-orange-500" />
                        ) : (
                          <UserCheck className="w-4 h-4 text-green-500" />
                        )}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        title="Edit user"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleEdit(user);
                        }}
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        title="Delete user"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteUser(user);
                        }}
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
              onPerPageChange={(v) => {
                setPerPage(v);
                setPage(1);
              }}
            />
          </div>
        )}
      </Card>

      {/* User Form Modal */}
      <UserFormModal
        isOpen={showFormModal}
        onClose={() => {
          setShowFormModal(false);
          setEditingUser(null);
        }}
        user={editingUser}
      />

      {/* Delete Confirmation */}
      <ConfirmModal
        isOpen={!!deleteUser}
        onClose={() => setDeleteUser(null)}
        onConfirm={() => deleteUser && deleteMutation.mutate(deleteUser.id)}
        title="Delete User"
        message={`Are you sure you want to delete "${deleteUser?.email}"? This action cannot be undone.`}
        confirmText="Delete"
        loading={deleteMutation.isPending}
      />

      {/* Status Change Confirmation */}
      <ConfirmModal
        isOpen={!!statusChangeUser}
        onClose={() => setStatusChangeUser(null)}
        onConfirm={() =>
          statusChangeUser &&
          updateStatusMutation.mutate({
            id: statusChangeUser.user.id,
            status: statusChangeUser.newStatus,
          })
        }
        title={statusChangeUser?.newStatus === 'ACTIVE' ? 'Activate User' : 'Deactivate User'}
        message={`Are you sure you want to ${statusChangeUser?.newStatus === 'ACTIVE' ? 'activate' : 'deactivate'} "${statusChangeUser?.user.email}"?`}
        confirmText={statusChangeUser?.newStatus === 'ACTIVE' ? 'Activate' : 'Deactivate'}
        loading={updateStatusMutation.isPending}
      />
    </div>
  );
}
