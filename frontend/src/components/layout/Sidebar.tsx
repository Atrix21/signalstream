import { Link, useLocation } from 'react-router-dom';
import { 
  HomeIcon, 
  KeyIcon, 
  BoltIcon, 
  BellIcon, 
  GlobeAltIcon 
} from '@heroicons/react/24/outline';
import clsx from 'clsx';

const navigation = [
  { name: 'Dashboard', href: '/', icon: HomeIcon },
  { name: 'Alerts', href: '/alerts', icon: BellIcon },
  { name: 'Strategies', href: '/strategies', icon: BoltIcon },
  { name: 'Events', href: '/events', icon: GlobeAltIcon },
  { name: 'API Keys', href: '/settings/keys', icon: KeyIcon },
];

export const Sidebar = () => {
  const location = useLocation();

  return (
    <div className="flex flex-col w-64 border-r border-tremor-border dark:border-dark-tremor-border bg-tremor-background dark:bg-dark-tremor-background h-screen sticky top-0">
      <div className="p-6">
        <h1 className="text-xl font-bold text-tremor-brand dark:text-dark-tremor-brand tracking-tight">
          SignalStream
        </h1>
      </div>
      
      <nav className="flex-1 px-4 space-y-1">
        {navigation.map((item) => {
          const isActive = location.pathname === item.href;
          return (
            <Link
              key={item.name}
              to={item.href}
              className={clsx(
                isActive
                  ? 'bg-tremor-brand-faint text-tremor-brand-emphasis dark:bg-dark-tremor-brand-faint dark:text-dark-tremor-brand-emphasis'
                  : 'text-tremor-content-emphasis hover:bg-tremor-background-muted dark:text-dark-tremor-content-emphasis dark:hover:bg-dark-tremor-background-muted',
                'group flex items-center px-2 py-2 text-sm font-medium rounded-tremor-default transition-colors'
              )}
            >
              <item.icon
                className={clsx(
                  isActive ? 'text-tremor-brand-emphasis dark:text-dark-tremor-brand-emphasis' : 'text-tremor-content-subtle dark:text-dark-tremor-content-subtle',
                  'mr-3 flex-shrink-0 h-6 w-6'
                )}
                aria-hidden="true"
              />
              {item.name}
            </Link>
          );
        })}
      </nav>

      <div className="p-4 border-t border-tremor-border dark:border-dark-tremor-border">
        <div className="flex items-center">
          <div className="ml-3">
            <p className="text-sm font-medium text-tremor-content-strong dark:text-dark-tremor-content-strong">
              User Account
            </p>
            <p className="text-xs text-tremor-content dark:text-dark-tremor-content">
              View Profile
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};
