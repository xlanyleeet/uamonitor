import { redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { api } from '#lib/server/api.ts';
import type { Server } from '#lib/types.ts';
import { serverState } from '#lib/presentation.ts';
export const GET: RequestHandler = async ({fetch,getClientAddress}) => {
  const servers=await api<Server[]>(fetch,'servers',getClientAddress());
  const online=servers.filter(s=>serverState(s.statuses[0])==='online');
  const pool=online.length?online:servers;
  redirect(303,pool.length?'/server/'+pool[Math.floor(Math.random()*pool.length)].id:'/?empty=1');
};
