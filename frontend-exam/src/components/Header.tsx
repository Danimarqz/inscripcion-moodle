import { useEffect, useState } from 'preact/hooks';
import { getAuthToken } from '../utils/adminAuth';

export default function Header() {
  const [authToken, setAuthToken] = useState<string | null>(null);

  useEffect(() => {
    setAuthToken(getAuthToken());
  }, []);


  return (
        <div id="admin-button-container">
          {authToken ? (
            <a href="/admin/dashboard" className="py-2 px-4 rounded-md font-semibold cursor-pointer transition-all duration-300 no-underline inline-block border-2 border-accent bg-transparent text-accent whitespace-nowrap hover:bg-accent hover:text-white hover:shadow-accent/30 hover:shadow-lg hover:-translate-y-0.5">
              Panel admin
            </a>
          ) : (
            <a href="/admin/login" className="py-2 px-4 rounded-md font-semibold cursor-pointer transition-all duration-300 no-underline inline-block border-2 border-accent bg-transparent text-accent whitespace-nowrap hover:bg-accent hover:text-white hover:shadow-accent/30 hover:shadow-lg hover:-translate-y-0.5">
              Admin Login
            </a>
          )}
        </div>
  );
}

