import { useEffect, useState } from 'preact/hooks';
import { getAuthToken } from './AdminDashboard';

export default function Header() {
  const [authToken, setAuthToken] = useState<string | null>(null);

  useEffect(() => {
    setAuthToken(getAuthToken());;
  }, []);


  return (
    <header>
      <div className="header-content">
        <a href="/" className="site-title">
          <h1>OpositaTest</h1>
        </a>
        <div id="admin-button-container">
          {authToken ? (
            <a href="/admin/dashboard" className="admin-button">Admin Dashboard</a>
          ) : (
            <a href="/admin/login" className="admin-button">Admin Login</a>
          )}
        </div>
      </div>
    </header>
  );
}
