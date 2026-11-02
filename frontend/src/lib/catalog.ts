import type { ParsedServer } from './types.ts';
import { serverState } from './presentation.ts';
export type Sort = 'players-desc' | 'players-asc' | 'name-asc' | 'name-desc';
export function filterServers(servers: ParsedServer[], query: string, state: string, sort: Sort, now = Date.now()): ParsedServer[] {
  const q = query.trim().toLocaleLowerCase('uk-UA');
  return servers.filter(s => (!q || [s.name,s.ip,s.statuses[0]?.version || ''].some(v => v.toLocaleLowerCase('uk-UA').includes(q))) &&
    (state === 'all' || serverState(s.statuses[0], now) === state)).sort((a,b) => {
      let result = 0;
      if (sort.startsWith('name')) result = a.name.localeCompare(b.name, 'uk');
      else result = (serverState(a.statuses[0], now) === 'online' ? a.statuses[0]?.onlineCount ?? 0 : 0) -
        (serverState(b.statuses[0], now) === 'online' ? b.statuses[0]?.onlineCount ?? 0 : 0);
      return (sort.endsWith('desc') ? -result : result) || a.id-b.id;
    });
}
