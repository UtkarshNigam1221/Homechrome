import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { Pagination } from '../Pagination';

describe('Pagination', () => {
  const base = {
    hasMore: false,
    hasPrevious: false,
    perPage: 100,
    onNextPage: vi.fn(),
    onPreviousPage: vi.fn(),
  };

  // The bug this guards: picking 100/page collapses the list to one page, and
  // hiding the whole bar then left no way back to 10.
  it('keeps the per-page select when a single page holds everything', () => {
    render(<Pagination {...base} onPerPageChange={vi.fn()} />);

    expect(screen.getByLabelText('Show:')).toHaveValue('100');
    expect(screen.queryByText('Next')).not.toBeInTheDocument();
  });

  it('renders nothing when there is no paging and no per-page select', () => {
    const { container } = render(<Pagination {...base} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('shows the nav buttons once there is a page to go to', () => {
    render(<Pagination {...base} hasMore onPerPageChange={vi.fn()} />);
    expect(screen.getByText('Next')).toBeInTheDocument();
  });
});
