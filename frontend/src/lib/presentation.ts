import type { Server, Status } from './types.ts';
export type ServerState = 'online' | 'offline' | 'stale' | 'unknown';
export const stateLabel: Record<ServerState, string> = { online: 'Онлайн', offline: 'Офлайн', stale: 'Застарілі дані', unknown: 'Ще не перевірено' };
export function serverState(status: Status | undefined, now = Date.now()): ServerState {
  if (!status) return 'unknown';
  if (now - Date.parse(status.date) > 20 * 60000) return 'stale';
  return status.isOnline ? 'online' : 'offline';
}
export function address(server: Pick<Server, 'ip' | 'port'>): string {
  const host = server.ip.includes(':') ? '[' + server.ip + ']' : server.ip;
  return host + (server.port === '25565' ? '' : ':' + server.port);
}
export function iconURL(icon: string | null): string {
  return icon?.startsWith('data:image/png;base64,') && icon.length <= 65536 ? icon : '/favicon.png';
}
export function httpURL(value: string | null): string | null {
  if (!value) return null;
  try { const u = new URL(value); return ['http:', 'https:'].includes(u.protocol) && !u.username && !u.password ? u.href : null; } catch { return null; }
}
export function timeLabel(value: string, full = false): string {
  return new Date(value).toLocaleString('uk-UA', { timeZone: 'Europe/Kyiv', ...(full ? { day: '2-digit', month: '2-digit' } : {}), hour: '2-digit', minute: '2-digit' });
}
