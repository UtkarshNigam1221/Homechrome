import { clsx } from 'clsx';
import { ChevronDown, ChevronsUpDown, ChevronUp } from 'lucide-react';

// Table Root
export interface TableProps extends React.HTMLAttributes<HTMLTableElement> {
  children: React.ReactNode;
}

export function Table({ children, className, ...props }: TableProps) {
  return (
    <div className="overflow-x-auto rounded-2xl border border-gray-100">
      <table className={clsx('min-w-full divide-y divide-gray-100', className)} {...props}>
        {children}
      </table>
    </div>
  );
}

// Table Header
export interface TableHeaderProps extends React.HTMLAttributes<HTMLTableSectionElement> {
  children: React.ReactNode;
}

export function TableHeader({ children, className, ...props }: TableHeaderProps) {
  return (
    <thead className={clsx('bg-gray-50/80', className)} {...props}>
      {children}
    </thead>
  );
}

// Table Body
export interface TableBodyProps extends React.HTMLAttributes<HTMLTableSectionElement> {
  children: React.ReactNode;
}

export function TableBody({ children, className, ...props }: TableBodyProps) {
  return (
    <tbody className={clsx('bg-white divide-y divide-gray-100', className)} {...props}>
      {children}
    </tbody>
  );
}

// Table Row
export interface TableRowProps extends React.HTMLAttributes<HTMLTableRowElement> {
  children: React.ReactNode;
  clickable?: boolean;
}

export function TableRow({ children, className, clickable, ...props }: TableRowProps) {
  return (
    <tr
      className={clsx(
        'hover:bg-gray-50/80 transition-all duration-150',
        clickable && 'cursor-pointer',
        className
      )}
      {...props}
    >
      {children}
    </tr>
  );
}

// Table Head Cell
export interface TableHeadProps extends React.ThHTMLAttributes<HTMLTableCellElement> {
  children: React.ReactNode;
  sortable?: boolean;
  sortDirection?: 'asc' | 'desc' | null;
  onSort?: () => void;
}

export function TableHead({
  children,
  className,
  sortable,
  sortDirection,
  onSort,
  ...props
}: TableHeadProps) {
  return (
    <th
      className={clsx(
        'px-6 py-4 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider',
        sortable &&
          'cursor-pointer select-none hover:bg-gray-100/80 transition-colors duration-150',
        className
      )}
      onClick={sortable ? onSort : undefined}
      {...props}
    >
      <div className="flex items-center gap-1">
        {children}
        {sortable && (
          <span className="ml-1">
            {sortDirection === 'asc' && <ChevronUp className="w-4 h-4" />}
            {sortDirection === 'desc' && <ChevronDown className="w-4 h-4" />}
            {!sortDirection && <ChevronsUpDown className="w-4 h-4 text-gray-300" />}
          </span>
        )}
      </div>
    </th>
  );
}

// Table Cell
export interface TableCellProps extends React.TdHTMLAttributes<HTMLTableCellElement> {
  children: React.ReactNode;
}

export function TableCell({ children, className, ...props }: TableCellProps) {
  return (
    <td className={clsx('px-6 py-4 whitespace-nowrap text-sm text-gray-700', className)} {...props}>
      {children}
    </td>
  );
}

// Empty State
export interface TableEmptyProps {
  message?: string;
  description?: string;
  action?: React.ReactNode;
  colSpan?: number;
}

export function TableEmpty({
  message = 'No data found',
  description,
  action,
  colSpan = 5,
}: TableEmptyProps) {
  return (
    <tr>
      <td colSpan={colSpan} className="px-6 py-12 text-center">
        <p className="text-sm font-medium text-gray-900">{message}</p>
        {description && <p className="mt-1 text-sm text-gray-500">{description}</p>}
        {action && <div className="mt-4">{action}</div>}
      </td>
    </tr>
  );
}

// Loading State
export interface TableLoadingProps {
  rows?: number;
  colSpan?: number;
}

export function TableLoading({ rows = 5, colSpan = 5 }: TableLoadingProps) {
  return (
    <>
      {Array.from({ length: rows }).map((_, i) => (
        <tr key={i} className="border-b border-gray-50">
          {Array.from({ length: colSpan }).map((_, j) => (
            <td key={j} className="px-6 py-4">
              <div className="h-4 bg-gray-100 rounded-lg animate-pulse" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}
