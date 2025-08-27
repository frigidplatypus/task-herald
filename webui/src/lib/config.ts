import { writable } from 'svelte/store';
import { browser } from '$app/environment';

export type SortDirection = 'asc' | 'desc';
export interface Config {
  columns: string[];
  sort: { key: string; direction: SortDirection };
}

// Load persisted config or fallback to defaults
const defaultConfig: Config = { columns: [], sort: { key: '', direction: 'asc' } };
const stored = browser ? localStorage.getItem('taskherald-config') : null;
const initialConfig: Config = stored ? JSON.parse(stored) : defaultConfig;

export const config = writable<Config>(initialConfig);

// Persist config on changes
if (browser) {
  config.subscribe((val) => {
    localStorage.setItem('taskherald-config', JSON.stringify(val));
  });
}
