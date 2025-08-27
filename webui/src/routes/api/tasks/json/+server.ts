// SvelteKit endpoint to proxy /api/tasks/json requests to the Go backend
import type { RequestHandler } from '@sveltejs/kit';

export const GET: RequestHandler = async ({ fetch, url }) => {
  // The Go backend is expected to run on localhost:8080
  const backendUrl = 'http://localhost:8080/api/tasks/json';
  const backendRes = await fetch(backendUrl);
  const body = await backendRes.text();
  return new Response(body, {
    status: backendRes.status,
    headers: {
      'content-type': backendRes.headers.get('content-type') || 'application/json'
    }
  });
};
