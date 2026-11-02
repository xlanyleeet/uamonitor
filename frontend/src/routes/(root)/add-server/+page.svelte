<script lang="ts">
  import Icon from '#lib/components/Icon.svelte';
  import { tick, onMount } from 'svelte';
  let enabled=false; let busy=false; let applicationID=''; let requestKey='';
  onMount(()=>{
    const bytes=crypto.getRandomValues(new Uint8Array(32));requestKey=Array.from(bytes,b=>b.toString(16).padStart(2,'0')).join('');
    try {
      const draft=JSON.parse(sessionStorage.getItem('um-application-draft') || 'null');
      if (draft && draft.version===1 && Date.now()-draft.savedAt<86400000 && draft.savedAt<=Date.now()) {
        const values=draft.values;
        const fields=['name','host','port','versions','modes','description','website','community','owner','role','channel','contact','rules','language','access','accessDetails','note'];
        if (values && fields.every(key=>typeof values[key]==='string' && values[key].length<=2000) && /^[a-f0-9]{64}$/.test(draft.requestKey)) {
          ({name,host,port,versions,modes,description,website,community,owner,role,channel,contact,rules,language,access,accessDetails,note}=values);
          requestKey=draft.requestKey;
          draftMessage='Чернетку відновлено. Перевірте дані перед надсиланням.';
        }
      }
    } catch {}
    draftReady=true;
    void (async()=>{try{const r=await fetch('/api/applications/config');enabled=r.ok&&(await r.json()).enabled;}catch{}})();
  });
  let step = 1;
  let complete = false;
  let form: HTMLFormElement;
  let name = '';
  let host = '';
  let port = '25565';
  let versions = '';
  let modes = '';
  let description = '';
  let website = '';
  let community = '';
  let owner = '';
  let role = 'owner';
  let channel = 'telegram';
  let contact = '';
  let rules = '';
  let language = '';
  let access = 'open';
  let accessDetails = '';
  let note = '';
  let authorized = false;
  let accepted = false;
  let message = '';
  let heading: HTMLHeadingElement;
  let draftReady=false;
  let draftMessage='';
  let statusLink='';
  $: if (draftReady && !complete) {
    const values={name,host,port,versions,modes,description,website,community,owner,role,channel,contact,rules,language,access,accessDetails,note};
    try { sessionStorage.setItem('um-application-draft',JSON.stringify({version:1,savedAt:Date.now(),requestKey,values})); }
    catch { draftMessage='Збереження чернетки в цій вкладці недоступне.'; }
  }
  let copyMessage = '';
  async function copyApplicationID() {
    try {
      await navigator.clipboard.writeText(applicationID);
      copyMessage = 'Номер скопійовано';
    } catch {
      copyMessage = 'Не вдалося скопіювати. Виділіть номер і скопіюйте вручну.';
    }
  }
  const steps = ['Про сервер','Вимоги й контакт','Перевірка даних'];
  function validatePort(event: Event) {
    const input=event.currentTarget as HTMLInputElement;
    const n=Number(input.value);
    input.setCustomValidity(n>=1&&n<=65535?'':'Порт має бути від 1 до 65535.');
  }
  async function advance() {
    if (!form.reportValidity()) return;
    message='';
    if (step===1) {
      const raw=host.trim();
      if (raw.includes('://') || /[\s/]/.test(raw)) {message='Вкажіть лише домен або IP, без https:// та шляху.';return;}
      step=2;
    } else if (step===2) {
      if (channel==='email' && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(contact.trim())) {message='Перевірте email для відповіді.';return;}
      step=3;
    } else {
      if(busy || !enabled) return;
      busy=true;
      try {
        const response=await fetch('/api/applications',{method:'POST',headers:{'Content-Type':'application/json','Idempotency-Key':requestKey},body:JSON.stringify({name,host,port,versions,modes,description,website,community,owner,role,channel,contact,rules,language,access,accessDetails,note,authorized,accepted})});
        const result=await response.json();
        if(!response.ok){message=result.message || 'Не вдалося надіслати заявку.';return;}
        applicationID=result.id;complete=true;
        statusLink=window.location.origin+'/applications/'+encodeURIComponent(applicationID)+'#'+requestKey;
        try { sessionStorage.removeItem('um-application-draft'); } catch {}
      }catch{message='Немає зв’язку із сервером. Повторіть надсилання: повторний запит не створить дубль.';}finally{busy=false;}
    }
    await tick();
    heading?.focus();
  }
  function back() { step=Math.max(1,step-1);message='';heading?.focus(); }
</script>

<svelte:head><title>Заявка на додавання — Ґрадієнт · UAMonitor</title><meta name="robots" content="noindex" /></svelte:head>
<div class="admission-banner"><span class="admission-demo-dot"></span><strong>Ручна перевірка</strong><span>{enabled?'Заявка зберігається й надсилається до приватного каналу модерації в Discord.':'Приймання заявок ще не підключено. Ви можете переглянути форму.'}</span></div>
<a class="back-link" href="/requirements"><Icon name="arrow" size={16} />Вимоги до серверів</a>

{#if complete}
  <section class="application-complete panel" aria-labelledby="application-complete-heading">
    <div class="application-complete-heading">
      <div class="application-complete-icon"><Icon name="check" size={26} /></div>
      <div><span class="tag">ЗАЯВКУ ЗБЕРЕЖЕНО</span><h1 id="application-complete-heading" bind:this={heading} tabindex="-1">Дякуємо за заявку.</h1></div>
    </div>
    <div class="application-complete-details">
      <section aria-label="Дані заявки">
        <h2 class="application-complete-server">{name}</h2>
        <p class="application-complete-status"><span aria-hidden="true"></span>Очікує перевірки</p>
        <p class="application-complete-description">Модератор перевірить дані сервера за вимогами каталогу. За потреби зв’яжеться з вами через вказаний контакт.</p>
        <p class="application-complete-description">Для відповіді: <strong>{channel==='email'?'Email':channel==='telegram'?'Telegram':'Discord'} · {contact}</strong></p>
        <div class="application-complete-id">
          <span class="small muted">Номер заявки</span>
          <div class="application-complete-id-row"><code>{applicationID}</code><button class="icon-button" type="button" aria-label="Скопіювати номер заявки" on:click={copyApplicationID}><Icon name={copyMessage === 'Номер скопійовано' ? 'check' : 'copy'} size={16} /></button></div>
          <p class="application-complete-feedback" role="status">{copyMessage}</p>
        </div>
      </section>
      <section class="application-next" aria-labelledby="application-next-heading">
        <h2 id="application-next-heading">Що далі?</h2>
        <ol>
          <li><span class="application-next-number" aria-hidden="true">1</span><div><strong>Перевірка модератором</strong><p>Звіримо дані з вимогами каталогу.</p></div></li>
          <li><span class="application-next-number" aria-hidden="true">2</span><div><strong>Уточнення за потреби</strong><p>Напишемо на вказаний вами контакт.</p></div></li>
          <li><span class="application-next-number" aria-hidden="true">3</span><div><strong>Поява в моніторингу</strong><p>Сервер з’явиться після схвалення.</p></div></li>
        </ol>
      </section>
    </div>
    <div class="application-complete-actions"><p>Збережіть номер заявки для звернення до адміністрації.</p><a class="button button-primary" href="/">До каталогу<Icon name="chevron" size={16} /></a></div>
    <div class="application-status-link"><a class="button" href={statusLink}>Переглянути статус заявки<Icon name="chevron" size={16} /></a><p class="small muted">Збережіть це приватне посилання: воно дає доступ до статусу та причини рішення.</p></div>
    <p class="application-complete-delivery"><Icon name="clock" size={16} /><span>Повідомлення модератору надійде через Discord. У разі збою надсилання повториться.</span></p>
  </section>
{:else}
  <p class="small muted application-draft-note" role="status">{draftMessage || 'Чернетка зберігається в цій вкладці на добу й не переноситься на інші пристрої.'}</p>
  <section class="application-heading"><div><p class="eyebrow">ҐРАДІЄНТ · ДОЛУЧИТИСЯ ДО КАТАЛОГУ</p><h1 bind:this={heading} tabindex="-1">Розкажіть про свій сервер.</h1><p class="intro-description">Кілька деталей, щоб познайомитися з вашим проєктом.<br />Публікація — після перевірки, за однаковими вимогами для всіх.</p></div></section>
  <div class="application-layout"><div><ol class="application-stepper" aria-label="Кроки заявки">{#each steps as label,index}<li class:current={step===index+1} class:finished={step>index+1} aria-current={step===index+1?'step':undefined}><span>{#if step>index+1}<Icon name="check" size={14} />{:else}{index+1}{/if}</span><span>{label}</span></li>{/each}</ol>
    <form bind:this={form} on:submit|preventDefault={advance} class="panel application-form" novalidate>
      {#if step===1}<fieldset><legend>Інформація про сервер</legend><p class="form-section-description">Ці дані з’являться у профілі після схвалення. Поля з * обов’язкові.</p>
        <label class="application-field">Назва сервера *<input bind:value={name} required minlength="2" maxlength="100" placeholder="Наприклад, Ґрадієнт" autocomplete="off" /></label>
        <div class="application-field-row address-fields"><label class="application-field">Адреса сервера *<input bind:value={host} required maxlength="253" placeholder="play.example.org" spellcheck="false" /><small>Домен або IP. Порт вкажіть окремо.</small></label><label class="application-field">Порт *<input bind:value={port} required type="text" inputmode="numeric" pattern="[0-9]+" maxlength="5" on:input={validatePort} name="port" /></label></div>
        <div class="application-field-row"><label class="application-field">Версії Minecraft *<input bind:value={versions} required maxlength="100" placeholder="Наприклад, 1.21.1–1.21.11" /></label><label class="application-field">Режими гри *<input bind:value={modes} required maxlength="200" placeholder="Виживання, мініігри, Skyblock…" /></label></div>
        <label class="application-field">Короткий опис *<textarea bind:value={description} required minlength="30" maxlength="1000" rows="4" placeholder="Що це за проєкт, для кого він і чим цікавий?"></textarea><small>{description.length}/1000 символів · мінімум 30</small></label>
        <label class="application-field">Вебсайт<input bind:value={website} type="url" placeholder="https://…" maxlength="500" /><small>Необов’язково. Власний сайт не є вимогою.</small></label>
        <label class="application-field">Публічна спільнота або підтримка *<input bind:value={community} required type="url" placeholder="Посилання на Discord, Telegram або сторінку підтримки" maxlength="500" /><small>Куди гравець може звернутися до адміністрації.</small></label>
      </fieldset>{:else if step===2}<fieldset><legend>Відповідність і контакт</legend><p class="form-section-description">Контакт представника — для модератора. Він не стане публічним посиланням у профілі.</p>
        <div class="application-field-row"><label class="application-field">Ваш нік або ім’я *<input bind:value={owner} required minlength="2" maxlength="100" placeholder="Як до вас звертатися" autocomplete="nickname" /></label><label class="application-field">Ваша роль<select bind:value={role}><option value="owner">Власник сервера</option><option value="admin">Представник адміністрації</option></select></label></div>
        <div class="application-field-row"><label class="application-field">Канал для відповіді<select bind:value={channel}><option value="telegram">Telegram</option><option value="discord">Discord</option><option value="email">Email</option></select></label><label class="application-field">Контакт *<input bind:value={contact} required minlength="3" maxlength="200" placeholder={channel==='email'?'owner@example.org':channel==='telegram'?'@username':'Ваш Discord username'} /></label></div>
        <label class="application-field">Посилання на правила *<input bind:value={rules} required type="url" maxlength="500" placeholder="https://…" /><small>Сторінка або доступне повідомлення у спільноті.</small></label>
        <label class="application-field">Яка мова офіційної комунікації? *<textarea bind:value={language} required minlength="10" maxlength="500" rows="3" placeholder="Якою мовою доступні правила, оголошення й підтримка?"></textarea></label>
        <label class="application-field">Як приєднатися?<select bind:value={access}><option value="open">Відкритий вхід</option><option value="whitelist">Whitelist або попередня заявка</option><option value="paid">Платний доступ</option><option value="other">Інші умови</option></select></label>
        {#if access!=='open'}<label class="application-field">Поясніть умови входу *<textarea bind:value={accessDetails} required minlength="10" maxlength="1000" rows="3" placeholder="Що має зробити гравець? Як модератор може перевірити сервер?"></textarea></label>{/if}
        <label class="application-field">Коментар для модератора<textarea bind:value={note} maxlength="1000" rows="3" placeholder="Що ще потрібно знати перед перевіркою?"></textarea></label>
        <div class="application-confirmations"><label><input type="checkbox" required bind:checked={authorized} /><span>Я є власником сервера або маю дозвіл представляти його. *</span></label><label><input type="checkbox" required bind:checked={accepted} /><span>Я прочитав <a class="text-link" href="/requirements" target="_blank" rel="noopener noreferrer">вимоги</a> та підтверджую правдивість введених даних у заявці. *</span></label></div>
      </fieldset>{:else}<section class="application-review"><h2>Усе правильно?</h2><p class="form-section-description">Перевірте дані перед надсиланням. Заявка та приватний контакт будуть доступні модераторам у Discord.</p><div class="application-review-server"><span class="brand-symbol"><Icon name="server" size={28} /></span><div><h3>{name}</h3><p>{host}:{port}</p></div><span class="tag">JAVA EDITION</span></div><dl><div><dt>Опис</dt><dd>{description}</dd></div><div><dt>Версії та режими</dt><dd>{versions} · {modes}</dd></div><div><dt>Сайт</dt><dd>{website || 'Не вказано'}</dd></div><div><dt>Спільнота</dt><dd>{community}</dd></div><div><dt>Правила</dt><dd>{rules}</dd></div><div><dt>Мова</dt><dd>{language}</dd></div><div><dt>Умови входу</dt><dd>{access==='open'?'Відкритий вхід':accessDetails}</dd></div><div><dt>Представник</dt><dd>{owner} · {role==='owner'?'власник':'адміністрація'}</dd></div><div><dt>Приватний контакт</dt><dd>{channel}: {contact}</dd></div>{#if note}<div><dt>Коментар</dt><dd>{note}</dd></div>{/if}</dl><div class="admission-banner"><Icon name="clock" size={16} /><span>Заявка очікуватиме ручної перевірки. Сервер не публікується автоматично.</span></div></section>{/if}
      {#if message}<p role="status" class="application-message">{message}</p>{/if}
      <div class="application-form-actions">{#if step>1}<button class="button" type="button" on:click={back}><Icon name="arrow" size={15} />Назад</button>{:else}<span class="small muted">Minecraft Java Edition</span>{/if}<button class="button button-primary" type="submit" disabled={busy || (step===3 && !enabled)}>{busy?'Надсилання…':step===3?'Надіслати заявку':'Далі'}<Icon name={step===3?'check':'chevron'} size={16} /></button></div>
    </form>
  </div><aside class="application-sidebar"><section class="panel"><span class="tag">ПРО ВІДБІР</span><h2>Спочатку знайомство. Потім публікація.</h2><p>Онлайн і популярність не замінюють перевірку. Ми дивимося на відповідність вимогам і прозорість інформації.</p><a class="text-link" href="/requirements">Прочитати вимоги ↗</a></section><section class="application-sidebar-note"><Icon name="clock" size={18} /><h3>Як це працює</h3><ul><li>Заявка очікує перевірки.</li><li>За потреби — уточнення від власника.</li><li>Схвалення або пояснення причини відмови.</li></ul></section><p class="small muted">Контакти доступні модераторам для перевірки заявки. Вони не публікуються в каталозі. Власнику відповідає модератор через вказаний контакт.</p></aside></div>
{/if}
