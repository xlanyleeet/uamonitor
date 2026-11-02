// Disposable local fixtures for visual QA. Only the reserved IDs below are touched.
import { execFileSync } from 'node:child_process';
import assert from 'node:assert/strict';
const ids=[2147483640,2147483641,2147483642,2147483643];
const run=(...args)=>execFileSync('docker',['compose','-p','uamonitor-go',...args],{encoding:'utf8'});
const sql=q=>execFileSync('docker',['compose','-p','uamonitor-go','exec','-T','db','psql','-U','uamonitor','-d','uamonitor','-v','ON_ERROR_STOP=1','-At'],{encoding:'utf8',input:q}).trim();
if(process.argv[2]==='add'){
  assert.equal(sql(`SELECT count(*) FROM servers WHERE id IN (${ids.join(',')})`),'0','Fixture IDs already exist; refusing to overwrite');
  const names=['Козацька Січ','Карпати SMP','CraftUA','Сонях Skyblock'];
  const counts=[128,47,0,12];
  let query='BEGIN;';
  for(let i=0;i<ids.length;i++){
    query+=`INSERT INTO servers(id,name,ip,port,description,link,discord,youtube) VALUES (${ids[i]},'${names[i]}','127.0.0.1',${25560+i},'## Вітаємо у спільноті\n\nВиживання, дружня атмосфера та пригоди разом.\n\n**Це тимчасовий тестовий сервер для перевірки дизайну, а не реальна адреса.**\n\n<img src=x onerror="alert(1)">','https://example.org','test-community','https://www.youtube.com/@fixture');`;
    for(let n=143;n>=0;n--){
      const players=i===2?0:Math.max(0,Math.round(counts[i]*(.6+.4*Math.cos(n/12))+Math.sin(n/4)*5));
      query+=`INSERT INTO statuses(server_id,is_online,online_count,max_online,version,date) VALUES (${ids[i]},${i!==2},${players},200,'Paper 1.21.4',now()-interval '${n*10+(i===3?120:0)} minutes');`;
    }
  }
  sql(query+'COMMIT;');
  console.log('Four temporary design fixtures added');
}else if(process.argv[2]==='remove'){
  sql(`DELETE FROM servers WHERE id IN (${ids.join(',')}) AND ip='127.0.0.1' AND description LIKE '%тимчасовий тестовий сервер%'`);
  console.log('Temporary design fixtures removed');
}else throw new Error('Use add or remove');
run('restart','backend');
