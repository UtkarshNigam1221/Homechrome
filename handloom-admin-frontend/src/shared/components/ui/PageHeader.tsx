import type { ReactNode } from 'react';

export interface PageHeaderProps {
  title: string;
  /** Optional supporting line. Accepts a node so callers can pass dynamic copy. */
  subtitle?: ReactNode;
  /** Right-aligned action(s) — typically a Button or button group. */
  action?: ReactNode;
}

/**
 * Standard list-page header: title + optional subtitle on the left, optional
 * action on the right, stacking on mobile. Replaces the repeated
 * `flex … justify-between` + `page-title`/`page-subtitle` block.
 */
export function PageHeader({ title, subtitle, action }: PageHeaderProps) {
  return (
    <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 className="page-title">{title}</h1>
        {subtitle != null && <p className="page-subtitle">{subtitle}</p>}
      </div>
      {action}
    </div>
  );
}
