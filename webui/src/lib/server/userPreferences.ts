// Utility for robust server-side user preferences persistence
// Only import and use this in server-side code (SvelteKit endpoints)
import fs from 'fs';
import path from 'path';

const PREFS_FILE = path.resolve(process.cwd(), 'user-preferences.json');

export type UserPreferences = Record<string, { columnOrder: string[]; sortOptions: any }>;

export function loadUserPreferences(): UserPreferences {
  try {
    return JSON.parse(fs.readFileSync(PREFS_FILE, 'utf-8'));
  } catch {
    return {};
  }
}

export function saveUserPreferences(prefs: UserPreferences) {
  // Write atomically to avoid corruption
  const tmpFile = PREFS_FILE + '.tmp';
  fs.writeFileSync(tmpFile, JSON.stringify(prefs, null, 2), 'utf-8');
  fs.renameSync(tmpFile, PREFS_FILE);
}
