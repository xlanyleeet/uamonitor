<script lang="ts">
  export let name = 'server';
  export let size = 18;
  const paths: Record<string, string> = {
    server: 'M4 3h16v7H4z M4 14h16v7H4z M7 6h.01 M7 17h.01 M10 6h7 M10 17h7',
    search: 'M21 21l-5-5 M18 10a8 8 0 1 1-16 0 8 8 0 0 1 16 0',
    copy: 'M8 8h12v13H8z M16 8V3H3v13h5',
    check: 'M5 12l4 4L20 5',
    sun: 'M12 2v2 M12 20v2 M2 12h2 M20 12h2 M5 5l1.5 1.5 M17.5 17.5L19 19 M5 19l1.5-1.5 M17.5 6.5L19 5 M16 12a4 4 0 1 1-8 0 4 4 0 0 1 8 0',
    moon: 'M20 14a8 8 0 0 1-10-10 8.5 8.5 0 1 0 10 10',
    plus: 'M12 5v14 M5 12h14',
    shuffle: 'M3 5h3l12 14h3 M17 15l4 4-4 4 M3 19h3l4-5 M14 10l4-5h3 M17 1l4 4-4 4',
    arrow: 'M19 12H5 M11 6l-6 6 6 6',
    chevron: 'M9 5l7 7-7 7',
    sort: 'M8 3v18 M4 7l4-4 4 4 M16 21V3 M12 17l4 4 4-4',
    external: 'M14 3h7v7 M21 3L10 14 M10 3H3v18h18v-7',
    users: 'M16 21v-3a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v3 M13 6a4 4 0 1 1-8 0 4 4 0 0 1 8 0 M22 21v-3a4 4 0 0 0-3-3.8 M16 2.2a4 4 0 0 1 0 7.6',
    clock: 'M12 7v5l3 2 M22 12a10 10 0 1 1-20 0 10 10 0 0 1 20 0',
    chart: 'M3 3v18h18 M7 14l4-4 4 3 5-7',
    code: 'M8 5l-6 7 6 7 M16 5l6 7-6 7 M14 3l-4 18',
    close: 'M5 5l14 14 M19 5L5 19'
  };
</script>
<svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d={paths[name] || paths.server} /></svg>
