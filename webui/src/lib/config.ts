import { writable } from 'svelte/store';

const browser = typeof window !== 'undefined';

export type SortDirection = 'asc' | 'desc';
export interface Config {
  columns: string[];
  sort: { key: string; direction: SortDirection };
}

const defaultConfig: Config = { columns: [], sort: { key: '', direction: 'asc' } };
export const config = writable<Config>(defaultConfig);

// Load config from backend on startup (browser only)
if (browser) {
  fetch('/api/user-preferences')
    .then((res) => res.ok ? res.json() : null)
    .then((prefs) => {
      if (prefs && prefs.columnOrder && prefs.sortOptions) {
        config.set({ columns: prefs.columnOrder, sort: prefs.sortOptions });
      }
    })
    .catch(() => {});

  // Persist config to backend on changes
  let skipFirst = true;
  config.subscribe((val) => {
    // Avoid sending default config on first load
    if (skipFirst) { skipFirst = false; return; }
    fetch('/api/user-preferences', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ columnOrder: val.columns, sortOptions: val.sort })
    });
  });
}
