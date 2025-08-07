import { useState } from 'preact/hooks';
import { adminLogin } from '../services/adminService';

export function saveAuthToken(token: string) {
  localStorage.setItem('admin_access_token', token);
}

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
    <main>
      <h1>Admin Login</h1>
      <form onSubmit={handleSubmit}>
        <div class="form-group">
          <label for="username">Usuario:</label>
          <input
            type="text"
            id="username"
            value={username}
            onInput={(e) => setUsername((e.target as HTMLInputElement).value)}
            required
          />
        </div>
        <div class="form-group">
          <label for="password">Contraseña:</label>
          <input
            type="password"
            id="password"
            value={password}
            onInput={(e) => setPassword((e.target as HTMLInputElement).value)}
            required
          />
        </div>
        <button type="submit" class="submit-button">Login</button>
        {errorMessage && (
          <p class="error-message">{errorMessage}</p>
        )}
      </form>
    </main>
  );
}
