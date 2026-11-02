import type { PageServerLoad } from './$types';
import { api } from '#lib/server/api.ts';
import type { GraphServer, ParsedServer, Server } from '#lib/types.ts';

export const load: PageServerLoad = async ({ fetch, getClientAddress }) => {
  const [servers, graphs] = await Promise.all([
    api<Server[]>(fetch, 'servers', getClientAddress()), api<GraphServer[]>(fetch, 'graph', getClientAddress())
  ]);
  const byId = new Map(graphs.map(graph => [graph.id, graph]));
  return { servers: servers.map(server => ({ ...server, graph: byId.get(server.id) || { id: server.id, data: [] } })) as ParsedServer[] };
};
