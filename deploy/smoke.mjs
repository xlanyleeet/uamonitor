import assert from 'node:assert/strict';
const base = process.env.UM_TEST_BASE || 'http://127.0.0.1:9013';
async function request(path, expected = 200) {
  const response = await fetch(base + path, { signal: AbortSignal.timeout(10000) });
  assert.equal(response.status, expected, path);
  return response;
}
assert.match(await (await request('/')).text(), /UAMonitor/);
await request('/about');
const servers = await (await request('/api/servers')).json();
const graphs = await (await request('/api/graph')).json();
assert.ok(Array.isArray(servers));
assert.deepEqual(servers.map(s => s.id), graphs.map(g => g.id));
for (const server of servers.slice(0, 3)) {
  assert.equal(typeof server.port, 'string');
  assert.ok(Array.isArray(server.statuses));
  assert.equal((await (await request('/api/servers/' + server.id)).json()).id, server.id);
  await request('/s/' + server.id);
  await request('/server/' + server.id);
}
await request('/api/graph/12junk', 400);
await request('/server/12junk', 400);
assert.match(await (await request('/licenses/artlist-MIT.txt')).text(), /MIT License/);
await request('/api/servers/2147483647', 404);
console.log(JSON.stringify({ base, servers: servers.length, status: 'passed' }));
