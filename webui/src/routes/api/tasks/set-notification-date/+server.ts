import type { RequestHandler } from '@sveltejs/kit';

// Proxy POST to Go backend for setting/clearing notification_date
export const POST: RequestHandler = async ({ request }) => {
  const data = await request.json();
  const uuid = data.uuid;
  const notification_date = data.notification_date;
  const backendUrl = 'http://localhost:8080/api/set-notification-date';
  // Send as form data to Go backend
  const form = new URLSearchParams();
  form.set('uuid', uuid);
  form.set('notification_date', notification_date || '');
  const backendRes = await fetch(backendUrl, {
    method: 'POST',
    body: form,
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  });
  const text = await backendRes.text();
  return new Response(text, { status: backendRes.status, headers: { 'content-type': backendRes.headers.get('content-type') || 'text/plain' } });
};
