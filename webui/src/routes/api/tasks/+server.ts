import type { RequestHandler } from '@sveltejs/kit';

export const GET: RequestHandler = async () => {
  const res = await fetch('http://localhost:8080/api/tasks');
  if (!res.ok) {
    return new Response('Failed to fetch tasks from backend', { status: 500 });
  }
  const html = await res.text();
  // Parse the HTML table rows and convert to JSON (for now, just return as text)
  // In production, update Go backend to return JSON directly
  return new Response(html, {
    headers: { 'Content-Type': 'text/html' }
  });
};
