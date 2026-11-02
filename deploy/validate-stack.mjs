// Local development validation only: inserts a reserved fixture, restarts db/backend,
// verifies persistence and removes the fixture in finally. Does not drop any volume.
import { execFileSync } from 'node:child_process';
import assert from 'node:assert/strict';
import { setTimeout as delay } from 'node:timers/promises';
const base = process.env.UM_TEST_BASE || 'http://127.0.0.1:9013';
const id = 2147483646;
const compose = (...args) => execFileSync('docker', ['compose', '-p', 'uamonitor-go', ...args], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] });
const sql = statement => compose('exec', '-T', 'db', 'psql', '-U', 'uamonitor', '-d', 'uamonitor', '-v', 'ON_ERROR_STOP=1', '-Atc', statement).trim();
async function ready(predicate = () => true) {
  for (let n = 0; n < 40; n++) {
    try {
      const response = await fetch(base + '/api/servers', { signal: AbortSignal.timeout(3000), cache: 'no-store' });
      if (response.ok && predicate(await response.json())) return;
    } catch {}
    await delay(500);
  }
  throw new Error('Stack did not become ready');
}
assert.equal(sql('SELECT count(*) FROM servers WHERE id=' + id), '0', 'reserved fixture ID is already in use');
let created = false;
try {
  sql(`INSERT INTO servers(id,name,ip,port,description,youtube) VALUES (${id},'Local validation fixture','127.0.0.1',25565,'**Validation** <img src=x onerror="window.auditXSS=1">','https://www.youtube.com/@fixture')`);
  created = true;
  sql(`INSERT INTO statuses(server_id,is_online,online_count,max_online,version) VALUES (${id},true,7,50,'Fixture 1.21')`);
  compose('restart', 'backend');
  await ready(rows => rows.some(s => s.id === id));
  await import('./smoke.mjs');
  const embed = await fetch(base + '/s/' + id);
  assert.equal(embed.status, 200);
  assert.match(await embed.text(), /Local validation fixture/);
  const originalCount = sql(`SELECT count(*) FROM statuses WHERE server_id=${id}`);
  compose('restart', 'db');
  await ready(rows => rows.some(s => s.id === id));
  assert.equal(sql(`SELECT count(*) FROM statuses WHERE server_id=${id}`), originalCount);
  assert.equal(sql(`SELECT online_count FROM statuses WHERE server_id=${id} ORDER BY date DESC LIMIT 1`), '7');
  console.log('Populated API, embed and PostgreSQL restart persistence: passed');
} finally {
  if (created) {
    sql('DELETE FROM servers WHERE id=' + id);
    compose('restart', 'backend');
    await ready(rows => !rows.some(s => s.id === id));
    console.log('Temporary validation fixture removed');
  }
}
