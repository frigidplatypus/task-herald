// Debug logging for persistence
function debug(...args: any[]) {
	// eslint-disable-next-line no-console
	console.log('[user-preferences]', ...args);
}
import type { RequestHandler } from '@sveltejs/kit';
import { loadUserPreferences, saveUserPreferences } from '$lib/server/userPreferences';


let userPreferences = loadUserPreferences();

export const GET: RequestHandler = async ({ locals }) => {
	const userId = locals.user?.id || 'default';
	userPreferences = loadUserPreferences();
	const prefs = userPreferences[userId] || { columnOrder: [], sortOptions: {} };
	debug('GET', { userId, prefs });
	return new Response(JSON.stringify(prefs), { status: 200 });
};

export const POST: RequestHandler = async ({ request, locals }) => {
	const userId = locals.user?.id || 'default';
	const { columnOrder, sortOptions } = await request.json();
	userPreferences = loadUserPreferences();
	userPreferences[userId] = { columnOrder, sortOptions };
	saveUserPreferences(userPreferences);
	debug('POST', { userId, columnOrder, sortOptions });
	return new Response(JSON.stringify({ success: true }), { status: 200 });
};
