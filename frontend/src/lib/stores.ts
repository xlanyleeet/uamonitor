import { writable } from 'svelte/store';
import { browser } from '$app/env';
let initial = 'light';
if (browser) {
  try { initial = new URLSearchParams(location.search).get('theme') || localStorage.getItem('theme') || 'light'; } catch {}
}
export const theme = writable(initial === 'dark' ? 'dark' : 'light');
if (browser) theme.subscribe(value => { try { localStorage.setItem('theme', value); } catch {} });
