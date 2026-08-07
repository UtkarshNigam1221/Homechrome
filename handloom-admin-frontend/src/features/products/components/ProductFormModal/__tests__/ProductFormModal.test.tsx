import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { type ReactNode, useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { categoriesApi } from '@/features/categories/api';
import { productsApi } from '@/features/products/api';

import type { Product } from '../../../types';
import { ProductFormModal } from '../ProductFormModal';

vi.mock('@/features/products/api', () => ({
  productsApi: {
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    adjustStock: vi.fn(),
  },
}));

vi.mock('@/features/categories/api', () => ({
  categoriesApi: {
    list: vi.fn(),
    getAttributes: vi.fn(),
  },
}));

const COLOUR_ATTR = {
  name: 'color',
  label: 'Colour',
  type: 'MULTI_SELECT' as const,
  required: false,
  searchable: true,
  display_order: 1,
  options: [
    { value: 'White', label: 'White' },
    { value: 'Green', label: 'Green' },
    { value: 'Blue', label: 'Blue' },
  ],
};

function makeProduct(attributes: Record<string, unknown>): Product {
  return {
    id: 'prod_1',
    name: 'Double Bed Polycot Bedsheet',
    sku: 'DBN1230',
    slug: 'double-bed-polycot-bedsheet',
    category_id: 'cat_1',
    base_price: 100000,
    selling_price: 90000,
    currency: 'INR',
    allow_custom_dimensions: false,
    attributes,
    quantity: 5,
    reserved_qty: 0,
    available_qty: 5,
    low_stock_threshold: 2,
    sort_order: 1,
    status: 'ACTIVE',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  };
}

// One client per test, not per render — the rerender-based cases below depend on
// the cache surviving across renders.
function Wrapper({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false } } })
  );
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

function colourCheckbox(label: string) {
  return screen.getByRole('checkbox', { name: label });
}

// jsdom does not perform implicit form submission from a submit-button click,
// so dispatch the submit event on the form itself.
function submitForm() {
  const form = screen.getByRole('button', { name: /update product/i }).closest('form');
  if (!form) throw new Error('product form not found');
  fireEvent.submit(form);
}

describe('ProductFormModal — MULTI_SELECT hydration', () => {
  beforeEach(() => {
    vi.mocked(categoriesApi.list).mockResolvedValue({
      items: [{ id: 'cat_1', name: 'Bedsheets' }],
      pagination: { limit: 100, has_more: false },
    } as never);
    vi.mocked(categoriesApi.getAttributes).mockResolvedValue({
      own_attributes: [COLOUR_ATTR],
      total_count: 1,
    });
  });

  it('pre-checks the colours already saved on the product', async () => {
    const product = makeProduct({ color: ['Green', 'White'] });
    vi.mocked(productsApi.get).mockResolvedValue(product);

    render(<ProductFormModal isOpen onClose={() => {}} product={product} />, { wrapper: Wrapper });

    await waitFor(() => expect(colourCheckbox('Green')).toBeInTheDocument());
    expect(colourCheckbox('Green')).toBeChecked();
    expect(colourCheckbox('White')).toBeChecked();
    expect(colourCheckbox('Blue')).not.toBeChecked();
  });

  it('pre-checks a single colour that loads as a plain string', async () => {
    const product = makeProduct({ color: 'Blue' });
    vi.mocked(productsApi.get).mockResolvedValue(product);

    render(<ProductFormModal isOpen onClose={() => {}} product={product} />, { wrapper: Wrapper });

    await waitFor(() => expect(colourCheckbox('Blue')).toBeInTheDocument());
    expect(colourCheckbox('Blue')).toBeChecked();
    expect(colourCheckbox('Green')).not.toBeChecked();
  });

  it('pre-checks colours when the modal is mounted closed then opened', async () => {
    const product = makeProduct({ color: ['Green', 'White'] });
    vi.mocked(productsApi.get).mockResolvedValue(product);

    const { rerender } = render(
      <ProductFormModal isOpen={false} onClose={() => {}} product={null} />,
      { wrapper: Wrapper }
    );
    rerender(<ProductFormModal isOpen onClose={() => {}} product={product} />);

    await waitFor(() => expect(colourCheckbox('Green')).toBeInTheDocument());
    expect(colourCheckbox('Green')).toBeChecked();
    expect(colourCheckbox('White')).toBeChecked();
  });

  it('pre-checks colours when only the detail fetch carries the attributes', async () => {
    // The list row may arrive without the attributes map; the detail fetch fills it in.
    const listProduct = makeProduct({});
    delete (listProduct as { attributes?: unknown }).attributes;
    vi.mocked(productsApi.get).mockResolvedValue(makeProduct({ color: ['Green', 'White'] }));

    render(<ProductFormModal isOpen onClose={() => {}} product={listProduct} />, {
      wrapper: Wrapper,
    });

    await waitFor(() => expect(colourCheckbox('Green')).toBeChecked());
    expect(colourCheckbox('White')).toBeChecked();
  });

  it('renders a saved colour that is missing from the category option list', async () => {
    // Real dev data has values (e.g. "Multicolour", material "COTTON") that the
    // category option list does not define. Those must still show up, checked.
    const product = makeProduct({ color: ['Green', 'Multicolour'] });
    vi.mocked(productsApi.get).mockResolvedValue(product);

    render(<ProductFormModal isOpen onClose={() => {}} product={product} />, { wrapper: Wrapper });

    await waitFor(() => expect(colourCheckbox('Green')).toBeChecked());
    expect(colourCheckbox('Multicolour')).toBeChecked();
  });

  it('submits an off-list colour back unchanged when the form is saved untouched', async () => {
    // The round trip is the real requirement: a value the definition does not
    // cover must survive a save, not just render. The repo replaces every
    // attribute row on update, so anything dropped from this payload is gone.
    const product = makeProduct({ color: ['Green', 'Multicolour'] });
    vi.mocked(productsApi.get).mockResolvedValue(product);
    vi.mocked(productsApi.update).mockResolvedValue(product);

    render(<ProductFormModal isOpen onClose={() => {}} product={product} />, { wrapper: Wrapper });

    await waitFor(() => expect(colourCheckbox('Multicolour')).toBeChecked());
    submitForm();

    await waitFor(() => expect(productsApi.update).toHaveBeenCalled());
    expect(productsApi.update).toHaveBeenCalledWith(
      'prod_1',
      expect.objectContaining({ attributes: { color: ['Green', 'Multicolour'] } })
    );
  });

  it('submits the remaining colours when an off-list colour is unchecked', async () => {
    const product = makeProduct({ color: ['Green', 'Multicolour'] });
    vi.mocked(productsApi.get).mockResolvedValue(product);
    vi.mocked(productsApi.update).mockResolvedValue(product);

    render(<ProductFormModal isOpen onClose={() => {}} product={product} />, { wrapper: Wrapper });

    await waitFor(() => expect(colourCheckbox('Multicolour')).toBeChecked());
    fireEvent.click(colourCheckbox('Multicolour'));
    submitForm();

    await waitFor(() => expect(productsApi.update).toHaveBeenCalled());
    expect(productsApi.update).toHaveBeenCalledWith(
      'prod_1',
      expect.objectContaining({ attributes: { color: ['Green'] } })
    );
  });

  it('pre-checks colours after the create form was used first', async () => {
    const product = makeProduct({ color: ['Green', 'White'] });
    vi.mocked(productsApi.get).mockResolvedValue(product);

    const { rerender } = render(<ProductFormModal isOpen onClose={() => {}} product={null} />, {
      wrapper: Wrapper,
    });
    // create form open, no product; then the user closes it and edits a product
    rerender(<ProductFormModal isOpen={false} onClose={() => {}} product={null} />);
    rerender(<ProductFormModal isOpen onClose={() => {}} product={product} />);

    await waitFor(() => expect(colourCheckbox('Green')).toBeInTheDocument());
    expect(colourCheckbox('Green')).toBeChecked();
    expect(colourCheckbox('White')).toBeChecked();
  });
});
