import { useAdminAuth } from '../hooks/useAdminAuth';
import { removeAuthToken, redirectToLogin } from '../utils/adminAuth';

type LinkItem = {
  href: string;
  title: string;
  description: string;
};

const ADMIN_LINKS: LinkItem[] = [
  {
    href: '/admin/exams',
    title: 'Gestionar examenes',
    description: 'Crear, editar o eliminar examenes disponibles.',
  },
  {
    href: '/admin/submissions',
    title: 'Gestionar intentos',
    description: 'Revisar y actualizar intentos enviados por los usuarios.',
  },
  {
    href: '/admin/results',
    title: 'Resultados oficiales',
    description: 'Importar PDFs oficiales y revisar su coincidencia con los usuarios.',
  },
];

export default function AdminDashboard() {
  const { token, loading, error } = useAdminAuth();

  void token;

  function handleLogout() {
    removeAuthToken();
    redirectToLogin();
  }

  if (loading) {
    return <p>Cargando panel de administracion...</p>;
  }

  return (
    <div className="max-w-5xl mx-auto my-8 p-8 bg-[#1a1c22] rounded-lg shadow-2xl text-white">
      <header className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-purple-300">Panel de administracion</h1>
          <p className="text-sm text-gray-400">Selecciona una seccion para comenzar a trabajar.</p>
        </div>
        <button
          className="bg-gray-700 border-none py-2 px-5 rounded text-white cursor-pointer transition-colors duration-300 hover:bg-gray-800"
          onClick={handleLogout}
        >
          Cerrar sesion
        </button>
      </header>

      {error && (
        <p className="text-red-500 bg-red-500/10 border border-red-500 p-4 rounded-md my-6">
          Error: {error}
        </p>
      )}

      <div className="grid gap-6 md:grid-cols-3">
        {ADMIN_LINKS.map((item) => (
          <a
            key={item.href}
            href={item.href}
            className="block bg-[#2a2d34] p-6 rounded-lg shadow-lg border border-transparent transition-colors duration-200 hover:-translate-y-1 hover:border-purple-500 hover:bg-[#32353f]"
          >
            <h2 className="text-xl font-semibold text-purple-200 mb-2">{item.title}</h2>
            <p className="text-sm text-gray-300">{item.description}</p>
          </a>
        ))}
      </div>
    </div>
  );
}
