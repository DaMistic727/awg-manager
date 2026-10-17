<!--
  Адрес tun-интерфейса (settings.fakeipTunAddr4) — секция дровера движка
  (StatusDrawer). Один адрес на оба tun-режима (fakeip-tun и policy-tun),
  поэтому секция вне карточек режимов и видна при любом режиме.

  Меняется только при выключенном режиме: бэкенд отвергает смену при живом
  OpkgTun (TUN_ADDR_LOCKED) и тот же признак отдаёт в status.tunAddrLocked.
-->
<script lang="ts">
  import { m } from '$lib/i18n';
  import { untrack } from 'svelte';
  import { Input } from '$lib/components/ui';
  import { notifications } from '$lib/stores/notifications';
  import { DEFAULT_TUN_ADDR } from './tunAddr';
  import type { SingboxRouterSettings, SingboxRouterStatus } from '$lib/types';

  interface Props {
    cfg: SingboxRouterSettings;
    status: SingboxRouterStatus | null;
    onPatch: (patch: Partial<SingboxRouterSettings>) => void | Promise<void>;
  }
  let { cfg, status, onPatch }: Props = $props();

  const locked = $derived(status?.tunAddrLocked === true);

  // Лёгкая проверка формы; последнее слово за бэкендом (DeriveTunDNS).
  const TUN_ADDR_RE = /^(\d{1,3}\.){3}\d{1,3}\/30$/;
  const draft = $state({ v: '' });
  // Черновик сбрасывается только при смене СОХРАНЁННОГО значения: cfg
  // перечитывается после каждого автосохранения дровера (как в PolicyTunCard).
  let lastSaved: string | undefined = $state();
  $effect(() => {
    const saved = cfg.fakeipTunAddr4 ?? '';
    if (saved !== untrack(() => lastSaved)) {
      lastSaved = saved;
      draft.v = saved;
    }
  });

  async function commit(raw: string): Promise<void> {
    const v = raw.trim() || DEFAULT_TUN_ADDR;
    if (v === (cfg.fakeipTunAddr4 || DEFAULT_TUN_ADDR)) {
      draft.v = cfg.fakeipTunAddr4 ?? '';
      return;
    }
    if (!TUN_ADDR_RE.test(v)) {
      notifications.error(m.sb_router_tun_addr_invalid());
      draft.v = cfg.fakeipTunAddr4 ?? '';
      return;
    }
    await onPatch({ fakeipTunAddr4: v });
    // Отказ бэкенда applyPatch показывает уведомлением — о нём говорит cfg.
    draft.v = cfg.fakeipTunAddr4 ?? '';
  }
</script>

<section class="sec">
  <label class="sec-cap" for="tun-addr">{m.sb_router_tun_addr_label()}</label>
  <Input
    id="tun-addr"
    value={draft.v}
    placeholder={DEFAULT_TUN_ADDR}
    disabled={locked}
    fullWidth
    oninput={(v) => (draft.v = v)}
    onchange={commit}
  />
  <p class="hint">{m.sb_router_tun_addr_hint()}</p>
</section>

<style>
  /* Повторяют .sec/.sec-cap/.hint дровера (scoped-стили родителя не
     проникают в дочерний компонент — как в PolicyTunCard). */
  .sec {
    padding: 14px var(--sp-4);
    border-bottom: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .sec-cap {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
  }
  .hint { margin: 0; font-size: 11.5px; color: var(--text-muted); line-height: 1.4; }
</style>
