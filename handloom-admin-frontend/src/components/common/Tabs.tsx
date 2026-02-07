import { clsx } from 'clsx';

export interface TabItem {
  id: string;
  label: string;
  icon?: React.ReactNode;
  badge?: string | number;
}

export interface TabsProps {
  tabs: TabItem[];
  activeTab: string;
  onChange: (tabId: string) => void;
  variant?: 'pills' | 'underline' | 'boxed';
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export function Tabs({
  tabs,
  activeTab,
  onChange,
  variant = 'pills',
  size = 'md',
  className,
}: TabsProps) {
  const sizes = {
    sm: 'text-xs py-1.5 px-3',
    md: 'text-sm py-2 px-4',
    lg: 'text-base py-2.5 px-5',
  };

  if (variant === 'pills') {
    return (
      <div
        className={clsx(
          'inline-flex space-x-1 bg-gray-100/80 p-1 border border-gray-200/50 rounded-xl',
          className
        )}
      >
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => onChange(tab.id)}
            className={clsx(
              'cursor-pointer rounded-lg font-medium transition-all duration-200 flex items-center gap-2',
              sizes[size],
              activeTab === tab.id
                ? 'bg-white text-primary-600 shadow-sm ring-1 ring-inset ring-gray-200/50'
                : 'text-gray-500 hover:text-gray-700 hover:bg-white/50'
            )}
          >
            {tab.icon}
            {tab.label}
            {tab.badge !== undefined && (
              <span
                className={clsx(
                  'ml-1 px-1.5 py-0.5 text-xs font-semibold rounded-md',
                  activeTab === tab.id
                    ? 'bg-primary-100 text-primary-700'
                    : 'bg-gray-200 text-gray-600'
                )}
              >
                {tab.badge}
              </span>
            )}
          </button>
        ))}
      </div>
    );
  }

  if (variant === 'underline') {
    return (
      <div className={clsx('border-b border-gray-200', className)}>
        <nav className="flex space-x-8" aria-label="Tabs">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => onChange(tab.id)}
              className={clsx(
                'whitespace-nowrap border-b-2 font-medium transition-all duration-200 flex items-center gap-2 -mb-px',
                sizes[size],
                activeTab === tab.id
                  ? 'border-primary-500 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              )}
            >
              {tab.icon}
              {tab.label}
              {tab.badge !== undefined && (
                <span
                  className={clsx(
                    'ml-1 px-2 py-0.5 text-xs font-semibold rounded-full',
                    activeTab === tab.id
                      ? 'bg-primary-100 text-primary-700'
                      : 'bg-gray-100 text-gray-600'
                  )}
                >
                  {tab.badge}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>
    );
  }

  // Boxed variant
  return (
    <div
      className={clsx(
        'inline-flex space-x-1 bg-white p-1 border border-gray-200 rounded-xl shadow-sm',
        className
      )}
    >
      {tabs.map((tab) => (
        <button
          key={tab.id}
          onClick={() => onChange(tab.id)}
          className={clsx(
            'cursor-pointer rounded-lg font-medium transition-all duration-200 flex items-center gap-2',
            sizes[size],
            activeTab === tab.id
              ? 'bg-primary-600 text-white shadow-sm shadow-primary-600/25'
              : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'
          )}
        >
          {tab.icon}
          {tab.label}
          {tab.badge !== undefined && (
            <span
              className={clsx(
                'ml-1 px-1.5 py-0.5 text-xs font-semibold rounded-md',
                activeTab === tab.id ? 'bg-white/20 text-white' : 'bg-gray-200 text-gray-600'
              )}
            >
              {tab.badge}
            </span>
          )}
        </button>
      ))}
    </div>
  );
}
