<script lang="ts">
  import { onMount } from 'svelte';
  import Icon from '#lib/components/Icon.svelte';
  let busy=true;
  let message='';
  let result: {id:string;status:string;reason:string;serverID:number|null}|null=null;
  let key='';
  let id='';
  const labels: Record<string,string>={pending:'Очікує перевірки',approved:'Заявку схвалено',rejected:'Заявку відхилено'};
  async function refresh() {
    busy=true;message='';
    try {
      const response=await fetch('/api/applications/'+encodeURIComponent(id),{headers:{Authorization:'Bearer '+key},cache:'no-store',signal:AbortSignal.timeout(10000)});
      if (!response.ok) { message=response.status===404?'Посилання недійсне. Відкрийте повне приватне посилання, збережене після надсилання заявки.':'Не вдалося отримати статус. Спробуйте ще раз.';return; }
      result=await response.json();
    } catch { message='Немає зв’язку із сервером. Спробуйте ще раз.'; }
    finally { busy=false; }
  }
  onMount(()=>{id=window.location.pathname.split('/').filter(Boolean).pop() || '';key=window.location.hash.slice(1);if(!/^[a-f0-9]{24}$/.test(id)||!/^[a-f0-9]{64}$/.test(key)){busy=false;message='Відкрийте повне приватне посилання зі сторінки підтвердження заявки.';return;}void refresh();});
</script>
<svelte:head><title>Статус заявки — UAMonitor</title><meta name="robots" content="noindex, nofollow" /><meta name="referrer" content="no-referrer" /></svelte:head>
<section class="application-complete panel">
  <span class="tag">ВАША ЗАЯВКА</span>
  <h1>Статус заявки</h1>
  <div class="application-status-content" aria-live="polite" aria-busy={busy}>
    {#if result}<h2>{labels[result.status] || 'Статус недоступний'}</h2><p class="small muted">Номер заявки: <code>{result.id}</code></p>
      {#if result.status==='pending'}<p>Модератор перевірить дані та за потреби зв’яжеться з вами через вказаний у заявці контакт.</p>{/if}
      {#if result.reason}<h2>Пояснення модератора</h2><p class="application-decision-reason">{result.reason}</p>{/if}
      {#if result.status==='approved' && result.serverID}<a class="button button-primary" href={'/server/'+result.serverID}>До сервера<Icon name="chevron" size={16} /></a>{/if}
    {:else if busy}<p>Отримуємо статус…</p>{/if}
    {#if message}<p role="status">{message}</p>{/if}
  </div>
  <div class="application-complete-actions"><a class="button" href="/">До каталогу</a>{#if key && id}<button class="button" type="button" disabled={busy} on:click={refresh}>{busy?'Оновлення…':'Оновити статус'}</button>{/if}</div>
</section>
