import { useAuthStore } from '../../store/authStore';
import { ArrowRightOnRectangleIcon } from '@heroicons/react/24/outline';

export const Header = () => {
  const { user, logout } = useAuthStore();

  return (
    <header className="bg-tremor-background dark:bg-dark-tremor-background border-b border-tremor-border dark:border-dark-tremor-border h-16 flex items-center justify-between px-6 sticky top-0 z-10">
      <div className="flex items-center">
        {/* Placeholder for breadcrumbs or page title if needed */}
        <h2 className="text-sm font-medium text-tremor-content-subtle dark:text-dark-tremor-content-subtle">
          Welcome back, <span className="text-tremor-content-strong dark:text-dark-tremor-content-strong">{user?.email}</span>
        </h2>
      </div>

      <div className="flex items-center space-x-4">
        <button
          onClick={logout}
          className="flex items-center text-sm text-tremor-content hover:text-red-500 transition-colors"
        >
          <ArrowRightOnRectangleIcon className="h-5 w-5 mr-1" />
          Logout
        </button>
      </div>
    </header>
  );
};
