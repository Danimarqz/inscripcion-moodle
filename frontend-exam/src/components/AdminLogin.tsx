import { useState } from 'preact/hooks';
import { adminLogin } from '../services/adminService';
import { saveAuthToken } from '../utils/adminAuth';

export default function AdminLogin() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [errorMessage, setErrorMessage] = useState('');

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setErrorMessage('');

    try {
      const response = await adminLogin({ username, password });
      saveAuthToken(response.access_token);
      window.location.href = '/admin/dashboard';
    } catch (error: any) {
      setErrorMessage(error.message || 'Error desconocido al iniciar sesión.');
    }
  };

  return (
    <main className="max-w-lg mx-auto my-8 p-8 bg-[#1a1c22] rounded-lg shadow-2xl text-white">
      <h1 className="text-4xl font-bold text-center mb-8 text-purple-300">Admin Login</h1>
      <form onSubmit={handleSubmit}>
        <div className="mb-6">
          <label htmlFor="username" className="block font-bold text-purple-500 mb-2">Usuario:</label>
          <input
            type="text"
            id="username"
            value={username}
            onInput={(e) => setUsername((e.target as HTMLInputElement).value)}
            required
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400 focus:ring-2 focus:ring-purple-400/50"
          />
        </div>
        <div className="mb-6">
          <label htmlFor="password" className="block font-bold text-purple-500 mb-2">Contraseña:</label>
          <input
            type="password"
            id="password"
            value={password}
            onInput={(e) => setPassword((e.target as HTMLInputElement).value)}
            required
            className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-purple-400 focus:ring-2 focus:ring-purple-400/50"
          />
        </div>
        <button type="submit" className="w-full py-3 cursor-pointer text-lg font-bold mt-4 rounded-md bg-purple-600 hover:bg-purple-700 transition-colors disabled:opacity-70 disabled:cursor-not-allowed">
          Login
        </button>
        {errorMessage && (
          <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md mt-6">{errorMessage}</p>
        )}
      </form>
    </main>
  );
}
