import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { once } from 'node:events';
import test from 'node:test';

test('SSR uses bulk API, handles embed IDs and backend failures', async t => {
  const seen = [];
  let fail = false;
  const servers = [7, 8].map(id => ({
    id, name: 'Fixture ' + id, ip: 'mc.example.org', port: '25565',
    description: '<img src=x onerror="window.auditXSS=1"> **Safe description**',
    icon: null, link: 'https://example.org', telegram: null, discord: null,
    youtube: 'https://www.youtube.com/@fixture',
    uptime: { days: 30, percent: id === 7 ? 99.8 : null, coverage: id === 7 ? 75 : 0 },
    statuses: [{ date: new Date().toISOString(), isOnline: true, onlineCount: id, maxOnline: 50, version: 'Test' }]
  }));
  const graphs = servers.map(s => ({ id: s.id, data: [{ date: new Date().toISOString(), isOnline: true, onlineCount: s.id }] }));
  const api = createServer((req, res) => {
    seen.push(req.url);
    res.setHeader('Content-Type', 'application/json');
    if (fail) { res.writeHead(503).end('{"message":"database secret"}'); return; }
    if (req.url === '/api/servers') { res.end(JSON.stringify(servers)); return; }
    if (req.url === '/api/graph') { res.end(JSON.stringify(graphs)); return; }
    const [, resource, id] = req.url.match(/^\/api\/(servers|graph)\/(\d+)$/) || [];
    const item = (resource === 'servers' ? servers : graphs).find(x => x.id === Number(id));
    if (item) res.end(JSON.stringify(item)); else res.writeHead(404).end('{}');
  });
  api.listen(0, '127.0.0.1'); await once(api, 'listening');
  process.env.API_BASE_URL = 'http://127.0.0.1:' + api.address().port;
  const { handler } = await import('../build/handler.js');
  const app = createServer((req, res) => handler(req, res));
  app.listen(0, '127.0.0.1'); await once(app, 'listening');
  t.after(async () => { app.closeAllConnections(); api.closeAllConnections(); await Promise.all([new Promise(r => app.close(r)), new Promise(r => api.close(r))]); });
  const base = 'http://127.0.0.1:' + app.address().port;
  const response = await fetch(base);
  assert.equal(response.status, 200);
  const html = await response.text();
  assert.match(html, /Fixture 7/);
  assert.match(html, /Fixture 8/);
  assert.match(html, /Аптайм за 30 днів/);
  assert.match(html, /99,8%/);
  assert.match(html, /Немає даних/);
  assert.deepEqual([...seen].sort(), ['/api/graph', '/api/servers']);
  assert.ok(!html.includes('<img src=x onerror='), 'description must never be emitted as raw HTML');
  seen.length = 0;
  const embed = await fetch(base + '/s/7?theme=dark');
  assert.equal(embed.status, 200);
  assert.match(await embed.text(), /Fixture 7/);
  assert.deepEqual([...seen].sort(), ['/api/graph/7', '/api/servers/7']);
  seen.length = 0;
  const profile = await fetch(base + '/server/7');
  assert.equal(profile.status,200);
  const profileHTML = await profile.text();
  assert.match(profileHTML,/Активність гравців/);
  assert.match(profileHTML,/Доступність сервера/);
  assert.match(profileHTML,/75% періоду/);
  assert.match(profileHTML,/https:\/\/www.youtube.com\/@fixture/);
  assert.deepEqual([...seen].sort(),['/api/graph/7','/api/servers/7']);
  assert.equal((await fetch(base + '/server/999')).status,404);
  assert.equal((await fetch(base + '/server/12junk')).status,400);
  const random = await fetch(base + '/random',{redirect:'manual'});
  assert.equal(random.status,303);
  assert.match(random.headers.get('location'),/^\/server\/(7|8)$/);
  assert.equal((await fetch(base + '/s/999')).status, 404);
  assert.equal((await fetch(base + '/s/12junk')).status, 400);
  fail = true;
  const failed = await fetch(base);
  assert.equal(failed.status, 503);
  assert.ok(!(await failed.text()).includes('database secret'));
});
