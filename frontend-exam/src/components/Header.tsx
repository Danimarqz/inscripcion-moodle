import { useAdminAuth } from '../hooks/useAdminAuth';

export default function Header() {
  const { token, loading } = useAdminAuth({ redirectOnMissing: false, validate: false });

  const href = token ? '/admin/dashboard' : '/admin/login';
  const label = loading ? 'Comprobando acceso...' : token ? 'Panel admin' : 'Admin Login';

  return (
    <div id="admin-button-container">
      <a
        href={href}
        className="py-2 px-4 rounded-md font-semibold cursor-pointer transition-all duration-300 no-underline inline-block border-2 border-accent bg-transparent text-accent whitespace-nowrap hover:bg-accent hover:text-white hover:shadow-accent/30 hover:shadow-lg hover:-translate-y-0.5"
      >
        {label}
      </a>
    </div>
  );
}
