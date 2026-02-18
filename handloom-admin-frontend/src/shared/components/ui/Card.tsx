import { clsx } from 'clsx';

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
  padding?: 'none' | 'sm' | 'md' | 'lg';
}

export function Card({ children, className, padding = 'md', ...props }: CardProps) {
  const paddingClasses = {
    none: '',
    sm: 'p-4',
    md: 'p-6',
    lg: 'p-8',
  };

  return (
    <div
      className={clsx(
        'bg-white rounded-2xl shadow-sm shadow-gray-200/50 border border-gray-100 transition-all duration-300 hover:shadow-md hover:shadow-gray-200/60',
        paddingClasses[padding],
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}

export interface CardHeaderProps extends React.HTMLAttributes<HTMLDivElement> {
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
}

export function CardHeader({ title, subtitle, action, className, ...props }: CardHeaderProps) {
  return (
    <div
      className={clsx(
        'flex items-start justify-between pb-5 border-b border-gray-100 mb-5',
        className
      )}
      {...props}
    >
      <div>
        <h3 className="text-lg font-semibold text-gray-900 tracking-tight">{title}</h3>
        {subtitle && <p className="mt-1 text-sm text-gray-500">{subtitle}</p>}
      </div>
      {action && <div>{action}</div>}
    </div>
  );
}

// Stat Card for Dashboard
export interface StatCardProps {
  title: string;
  value: string | number;
  change?: number;
  changeLabel?: string;
  icon?: React.ReactNode;
  trend?: 'up' | 'down' | 'neutral';
}

export function StatCard({ title, value, change, changeLabel, icon, trend }: StatCardProps) {
  const trendColors = {
    up: 'text-emerald-700 bg-emerald-50 ring-1 ring-inset ring-emerald-600/20',
    down: 'text-red-700 bg-red-50 ring-1 ring-inset ring-red-600/20',
    neutral: 'text-gray-700 bg-gray-100 ring-1 ring-inset ring-gray-500/10',
  };

  return (
    <Card className="hover:shadow-lg hover:shadow-gray-200/60">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-medium text-gray-500">{title}</p>
          <p className="mt-2 text-3xl font-bold text-gray-900 tracking-tight">{value}</p>
          {change !== undefined && (
            <div className="mt-3 flex items-center gap-2">
              <span
                className={clsx(
                  'inline-flex items-center px-2.5 py-1 rounded-lg text-xs font-semibold',
                  trendColors[trend || (change >= 0 ? 'up' : 'down')]
                )}
              >
                {change >= 0 ? '+' : ''}
                {change}%
              </span>
              {changeLabel && <span className="text-xs text-gray-500">{changeLabel}</span>}
            </div>
          )}
        </div>
        {icon && (
          <div className="p-3 rounded-xl bg-gradient-to-br from-primary-50 to-orange-50 text-primary-600 shadow-sm ring-1 ring-inset ring-primary-100">
            {icon}
          </div>
        )}
      </div>
    </Card>
  );
}
