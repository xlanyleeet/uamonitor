<script lang="ts">import { page } from '$app/state';</script>
<svelte:head><title>Помилка {page.status} — UAMonitor</title></svelte:head>
<div class="error-page"><p class="eyebrow">ПОМИЛКА {page.status}</p><h1>{page.status===404?'Сервер не знайдено':'Не вдалося завантажити сторінку'}</h1><p class="muted">{page.error?.message || 'Спробуйте оновити сторінку трохи пізніше.'}</p><a class="button button-primary" href="/">До списку серверів</a></div>
