import test from 'node:test';
import assert from 'node:assert/strict';
import { filterServers } from '../src/lib/catalog.ts';
import { address, httpURL, serverState } from '../src/lib/presentation.ts';

const now = Date.UTC(2026,9,27,10);
const server = (id,name,count,age=0,online=true) => ({id,name,ip:'play.'+id+'.example.org',port:'25565',statuses:[{date:new Date(now-age).toISOString(),isOnline:online,onlineCount:count,version:'Paper 1.21'}]});
const servers=[server(1,'Брама',12),server(2,'Атлас',28),server(3,'Старий',1000,60*60000),server(4,'Офлайн',null,0,false)];

test('search matches name, address and version without mutating source',()=>{
  assert.deepEqual(filterServers(servers,' АТЛАС ','all','players-desc',now).map(s=>s.id),[2]);
  assert.deepEqual(filterServers(servers,'play.1','all','name-asc',now).map(s=>s.id),[1]);
  assert.equal(filterServers(servers,'1.21','all','name-asc',now).length,4);
  assert.deepEqual(servers.map(s=>s.id),[1,2,3,4]);
});
test('filters distinguish stale, unknown, online and offline; stale count does not win ranking',()=>{
  assert.deepEqual(filterServers(servers,'','online','players-desc',now).map(s=>s.id),[2,1]);
  assert.deepEqual(filterServers(servers,'','stale','players-desc',now).map(s=>s.id),[3]);
  assert.deepEqual(filterServers(servers,'','offline','players-desc',now).map(s=>s.id),[4]);
  assert.deepEqual(filterServers(servers,'','all','players-desc',now).map(s=>s.id),[2,1,3,4]);
  assert.equal(serverState(undefined,now),'unknown');
});
test('addresses and external URLs remain safe',()=>{
  assert.equal(address({ip:'2001:db8::1',port:'25565'}),'[2001:db8::1]');
  assert.equal(address({ip:'2001:db8::1',port:'25566'}),'[2001:db8::1]:25566');
  assert.equal(httpURL('javascript:alert(1)'),null);
  assert.equal(httpURL('https://user:pass@example.org'),null);
  assert.equal(httpURL('https://example.org'),'https://example.org/');
});
