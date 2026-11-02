import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { api } from '#lib/server/api.ts';
import type { Server, GraphServer } from '#lib/types.ts';
export const load: PageServerLoad = async ({ fetch,params,getClientAddress }) => {
  if (!/^[1-9][0-9]*$/.test(params.server) || Number(params.server)>2147483647) error(400,'Некоректний ID сервера');
  const [server,graph] = await Promise.all([api<Server>(fetch,'servers/'+params.server,getClientAddress()),api<GraphServer>(fetch,'graph/'+params.server,getClientAddress())]);
  return {server:{...server,graph}};
};
