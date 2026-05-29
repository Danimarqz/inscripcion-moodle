import { useAdminAuth } from "../hooks/useAdminAuth";
import { adminLogout } from "../services/adminService";
import { removeAuthToken, redirectToLogin } from "../utils/adminAuth";

interface AdminDashboardProps {
  className?: string;
}

const baseContainerClass = "flex flex-col gap-3 items-stretch md:items-end";

export default function AdminDashboard({ className = "" }: AdminDashboardProps) {
  const { token, loading, error } = useAdminAuth();

  void token;

  const containerClass = [baseContainerClass, className].filter(Boolean).join(" ");

  async function handleLogout() {
    await adminLogout();
    removeAuthToken();
    redirectToLogin();
  }

  if (loading) {
    return (
      <div className={containerClass}>
        <p className="text-sm text-gray-400">Comprobando credenciales...</p>
      </div>
    );
  }

  return (
    <div className={containerClass}>
      {error && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md" role="alert">
          Error: {error}
        </p>
      )}
      <button
        type="button"
        className="self-end bg-gray-700 border-none py-2 px-5 rounded text-white cursor-pointer transition-colors duration-300 hover:bg-gray-800"
        onClick={handleLogout}
      >
        Cerrar sesión
      </button>
    </div>
  );
}
