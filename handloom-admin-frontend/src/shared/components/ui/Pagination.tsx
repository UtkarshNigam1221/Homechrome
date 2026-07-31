import { clsx } from 'clsx';
import { ChevronLeft, ChevronRight } from 'lucide-react';

export interface PaginationProps {
  hasMore: boolean;
  hasPrevious: boolean;
  perPage: number;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onPerPageChange?: (perPage: number) => void;
  showPerPage?: boolean;
  perPageOptions?: number[];
  itemCount?: number;
}

export function Pagination({
  hasMore,
  hasPrevious,
  perPage,
  onNextPage,
  onPreviousPage,
  onPerPageChange,
  showPerPage = true,
  perPageOptions = [10, 25, 50, 100],
  itemCount,
}: PaginationProps) {
  const canPage = hasMore || hasPrevious;

  // Per-page control stays when the list fits one page — else no way back to 10.
  if (!canPage && !(showPerPage && onPerPageChange)) return null;

  return (
    <div className="flex flex-col sm:flex-row items-center justify-between gap-4 py-4">
      <div className="flex items-center gap-4">
        {itemCount !== undefined && (
          <p className="text-sm text-gray-600">
            Showing <span className="font-medium">{itemCount}</span> results
          </p>
        )}
        {showPerPage && onPerPageChange && (
          <div className="flex items-center gap-2">
            <label htmlFor="per-page" className="text-sm text-gray-600">
              Show:
            </label>
            <select
              id="per-page"
              value={perPage}
              onChange={(e) => onPerPageChange(Number(e.target.value))}
              className="px-2 py-1 text-sm border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              {perPageOptions.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      {canPage && (
        <nav className="flex items-center gap-2">
          <button
            onClick={onPreviousPage}
            disabled={!hasPrevious}
            className={clsx(
              'flex items-center gap-1 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
              !hasPrevious ? 'text-gray-300 cursor-not-allowed' : 'text-gray-600 hover:bg-gray-100'
            )}
          >
            <ChevronLeft className="w-4 h-4" />
            Previous
          </button>
          <button
            onClick={onNextPage}
            disabled={!hasMore}
            className={clsx(
              'flex items-center gap-1 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
              !hasMore ? 'text-gray-300 cursor-not-allowed' : 'text-gray-600 hover:bg-gray-100'
            )}
          >
            Next
            <ChevronRight className="w-4 h-4" />
          </button>
        </nav>
      )}
    </div>
  );
}
