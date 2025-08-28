import type { RequestHandler } from '@sveltejs/kit';
import { readFileSync } from 'fs';
import { join } from 'path';

export const GET: RequestHandler = async () => {
  // Locate user's TaskWarrior config
  const home = process.env.HOME || process.env.USERPROFILE;
  const rcPath = join(home || '', '.config', 'task', 'taskrc');

  let content: string;
  try {
    content = readFileSync(rcPath, 'utf-8');
  } catch (e) {
    // No taskrc or unreadable: return empty list
    return new Response(JSON.stringify([]), {
      status: 200,
      headers: { 'Content-Type': 'application/json' }
    });
  }

  // Parse UDA definitions: look for uda.<name>.label and uda.<name>.type
  const lines = content.split(/\r?\n/);
  const names = new Set<string>();
  const labels = new Map<string, string>();
  for (const line of lines) {
    // strip comments
    const clean = line.split('#')[0].trim();
    let m = clean.match(/^uda\.([^.]+)\.label\s*=\s*(.*)$/);
    if (m) {
      names.add(m[1]);
      labels.set(m[1], m[2]);
      continue;
    }
    m = clean.match(/^uda\.([^.]+)\.type\s*=\s*(.*)$/);
    if (m) {
      names.add(m[1]);
    }
  }

  const udas = Array.from(names).map((name) => ({
    key: name,
    label: labels.get(name) || name
  }));

  return new Response(JSON.stringify(udas), {
    status: 200,
    headers: { 'Content-Type': 'application/json' }
  });
};
