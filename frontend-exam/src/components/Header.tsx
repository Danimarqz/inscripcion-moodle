import { useEffect, useState } from 'preact/hooks';
import { getAuthToken } from '../utils/adminAuth';

export default function Header() {
  const [authToken, setAuthToken] = useState<string | null>(null);

  useEffect(() => {
    setAuthToken(getAuthToken());
  }, []);


  return (
    <header className="bg-[#1a1c22] sticky top-0 z-10 py-4 px-6 md:px-8 border-b border-dark-300 shadow-lg mb-12">
      <div className="max-w-6xl mx-auto flex justify-between items-center">
        <a href="/" className="text-white no-underline">
          <h1 className="text-3xl md:text-4xl font-bold m-0 bg-gradient-to-r from-accent via-accent-light to-white bg-clip-text text-transparent bg-[length:400%_400%] bg-right-bottom transition-all duration-1000 ease-in-out hover:bg-left-top">
            Simulador
          </h1>
        </a>
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
      </div>
    </header>
  );
}

