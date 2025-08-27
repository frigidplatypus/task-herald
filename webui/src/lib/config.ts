import { writable, type Writable } from 'svelte/store';

const browser = typeof window !== 'undefined';

export type SortDirection = 'asc' | 'desc';
export interface Config {
  columns: string[];
  sort: { key: string; direction: SortDirection };
}

export interface ConfigState {
  loading: boolean;
  config: Config;
}

const defaultConfig: Config = { columns: [], sort: { key: '', direction: 'asc' } };
const initialState: ConfigState = { loading: true, config: defaultConfig };
export const config: Writable<ConfigState> = writable(initialState);

if (browser) {
  fetch('/api/user-preferences')
    .then((res) => res.ok ? res.json() : null)
    .then((prefs) => {
      if (prefs && prefs.columnOrder && prefs.sortOptions) {
        config.set({ loading: false, config: { columns: prefs.columnOrder, sort: prefs.sortOptions } });
      } else {
        config.set({ loading: false, config: defaultConfig });
      }
    })
    .catch(() => {
      config.set({ loading: false, config: defaultConfig });
    });

  // Persist config to backend on changes
  let skipFirst = true;
  config.subscribe((val) => {
    if (skipFirst) { skipFirst = false; return; }
    fetch('/api/user-preferences', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ columnOrder: val.config.columns, sortOptions: val.config.sort })
    });
  });
}
