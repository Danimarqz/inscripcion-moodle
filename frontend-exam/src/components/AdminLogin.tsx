import { useState } from "preact/hooks";
import { adminLogin } from "../services/adminService";
import { saveAuthToken } from "../utils/adminAuth";

interface AdminLoginProps {
  className?: string;
  redirectTo?: string;
}

export default function AdminLogin({ className = "", redirectTo = "/admin/dashboard" }: AdminLoginProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (event: Event) => {
    event.preventDefault();

    if (isSubmitting) {
      return;
    }

    setErrorMessage("");
    setIsSubmitting(true);

    try {
      const response = await adminLogin({ username, password });
      saveAuthToken(response.access_token);
      window.location.href = redirectTo;
    } catch (error: any) {
      setErrorMessage(error?.message ?? "Error desconocido al iniciar sesión.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className={["space-y-6", className].filter(Boolean).join(" ")}>
      <div>
        <label htmlFor="username" className="block font-bold text-brand-pink mb-2">
          Usuario
        </label>
        <input
          type="text"
          id="username"
          value={username}
          onInput={(event) => setUsername((event.target as HTMLInputElement).value)}
          required
          className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
        />
      </div>
      <div>
        <label htmlFor="password" className="block font-bold text-brand-pink mb-2">
          Contraseña
        </label>
        <input
          type="password"
          id="password"
          value={password}
          onInput={(event) => setPassword((event.target as HTMLInputElement).value)}
          required
          className="w-full px-3 py-2 rounded border border-[#444] bg-[#2a2d33] text-white focus:outline-none focus:border-brand-blue focus:ring-2 focus:ring-brand-blue/50"
        />
      </div>
      <button
        type="submit"
        disabled={isSubmitting}
        className="w-full py-3 cursor-pointer text-lg font-bold rounded-md bg-brand-pink text-dark-200 hover:bg-brand-yellow transition-colors disabled:opacity-70 disabled:cursor-not-allowed"
      >
        {isSubmitting ? "Accediendo…" : "Acceder"}
      </button>
      {errorMessage && (
        <p className="text-center text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md" role="alert">
          {errorMessage}
        </p>
      )}
    </form>
  );
}
