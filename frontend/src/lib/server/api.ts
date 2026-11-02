import { env } from 'node:process';
import { error } from '@sveltejs/kit';

export async function api<T>(fetcher: typeof fetch, endpoint: string, clientAddress: string): Promise<T> {
  const base = env.API_BASE_URL || 'http://127.0.0.1:8080';
  let response: Response;
  try {
    response = await fetcher(new URL('/api/' + endpoint, base), {
      signal: AbortSignal.timeout(7000), headers: { 'X-Real-IP': clientAddress }
    });
  } catch { error(503, 'Моніторинг тимчасово недоступний. Спробуйте пізніше.'); }
  if (response.status === 404) error(404, 'Сервер не знайдено');
  if (!response.ok) error(503, 'Не вдалося отримати дані моніторингу');
  return response.json() as Promise<T>;
}
