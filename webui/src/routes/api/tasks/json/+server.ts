// SvelteKit endpoint to proxy /api/tasks/json requests to the Go backend
import type { RequestHandler } from '@sveltejs/kit';

export const GET: RequestHandler = async ({ fetch, url }) => {
  // Proxy to Go backend at 127.0.0.1:8080
  const backendUrl = 'http://localhost:8080/api/tasks/json';
  try {
    const backendRes = await fetch(backendUrl);
    const body = await backendRes.text();
    return new Response(body, {
      status: backendRes.status,
      headers: {
        'content-type': backendRes.headers.get('content-type') || 'application/json'
      }
    });
  } catch (err: any) {
    // Return bad gateway if backend is unavailable
    return new Response(JSON.stringify({ error: err.message }), { status: 502, headers: { 'content-type': 'application/json' } });
  }
};
