import { defineMiddleware } from 'astro:middleware';

const API_URL = import.meta.env.PUBLIC_API_URL;

// Server-side guard for /admin/* pages: validate the httpOnly auth cookie
// against the backend before rendering, so a missing/expired session is
// redirected to the login page instead of flashing the admin UI client-side.
export const onRequest = defineMiddleware(async (context, next) => {
  const { pathname } = context.url;

  // Gate admin pages only; let the login page (and everything else) through.
  if (!pathname.startsWith('/admin') || pathname.startsWith('/admin/login')) {
    return next();
  }

  const token = context.cookies.get('admin_token')?.value;
  if (!token) {
    return context.redirect('/admin/login');
  }

  try {
    const res = await fetch(`${API_URL}/admin/check-token`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) {
      return context.redirect('/admin/login');
    }
  } catch {
    // API unreachable: fail closed for admin pages.
    return context.redirect('/admin/login');
  }

  return next();
});
