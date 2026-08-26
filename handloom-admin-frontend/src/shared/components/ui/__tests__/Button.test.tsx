import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { Button } from '../Button';

describe('Button', () => {
  // The bug this guards: HTML defaults a button inside a form to type="submit", so
  // every Cancel in the app submitted the form it sat in. On the coupon modal that
  // saved on Cancel and, because closing clears the selection first, saved as a
  // create — POST /admin/coupons, 409, coupon already exists.
  it('does not submit the form it sits in', () => {
    const onSubmit = vi.fn((e: React.FormEvent) => e.preventDefault());
    const onClose = vi.fn();

    render(
      <form onSubmit={onSubmit}>
        <Button onClick={onClose}>Cancel</Button>
      </form>
    );

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onClose).toHaveBeenCalledOnce();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('submits when asked to', () => {
    const onSubmit = vi.fn((e: React.FormEvent) => e.preventDefault());

    render(
      <form onSubmit={onSubmit}>
        <Button type="submit">Save</Button>
      </form>
    );

    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    expect(onSubmit).toHaveBeenCalledOnce();
  });

  it('defaults the rendered type to button', () => {
    render(<Button>Plain</Button>);
    expect(screen.getByRole('button', { name: 'Plain' })).toHaveAttribute('type', 'button');
  });
});
