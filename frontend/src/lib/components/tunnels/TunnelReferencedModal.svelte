<script lang="ts">
	import { Modal, Button } from '$lib/components/ui';
	import type { TunnelReferencedError } from '$lib/types';
	import { goto } from '$app/navigation';
	import { describeRouterReference, type RouterReference } from '$lib/utils/tunnelRefs';
	import { setMode } from '$lib/components/sb-router/modeStore';
	import { m } from '$lib/i18n';
	import { usageLevel } from '$lib/stores/settings';
	import { singboxRouter } from '$lib/stores/singboxRouter';
	import { isRoutingSubTabVisible, routingSubTabMinLevel, usageLevelLabel } from '$lib/types/usageLevel';

	interface Props {
		open: boolean;
		details: TunnelReferencedError | null;
		tunnelName?: string;
		entityLabel?: string;
		onclose: () => void;
	}

	let { open, details, tunnelName, entityLabel, onclose }: Props = $props();

	const PLANE_HREF: Record<RouterReference['plane'], string> = {
		tproxy: '/routing?tab=singbox',
		fakeip: '/routing?tab=fakeip'
	};

	// Вкладки sing-box на «Маршрутизации» видны с уровня 'expert' (FakeIP —
	// ещё и в режиме fakeip-tun). Ссылка на невидимую вкладку — тупик: Tabs не
	// находит id и остаётся на вкладке по умолчанию (F640). Тогда вместо ссылки
	// говорим, с какого уровня раздел доступен.
	const routerSettings = singboxRouter.settings;
	let tproxyVisible = $derived(isRoutingSubTabVisible($usageLevel, 'singboxRouter'));
	let planeVisible = $derived<Record<RouterReference['plane'], boolean>>({
		tproxy: tproxyVisible,
		fakeip: tproxyVisible || $routerSettings?.routingMode === 'fakeip-tun'
	});
	// Режим движка нужен до захода на вкладки sing-box — тот же праймер, что у
	// страницы «Маршрутизация».
	$effect(() => {
		if (open && !$routerSettings) void singboxRouter.reloadSettings();
	});

	// DNS-серверы и группы TProxy видны только в экспертном режиме; ?mode=
	// modeStore читает лишь при загрузке, поэтому режим ставим после перехода.
	async function openPlane(e: MouseEvent, plane: RouterReference['plane']) {
		e.preventDefault();
		onclose();
		await goto(PLANE_HREF[plane]);
		if (plane === 'tproxy') setMode('expert');
	}
</script>

{#snippet openLink(plane: RouterReference['plane'])}
	{#if planeVisible[plane]}
		<a href={PLANE_HREF[plane]} onclick={(e) => openPlane(e, plane)}>{m.tunnels_referenced_open()}</a>
	{:else}
		<span class="level-note">{m.tunnels_referenced_needs_level({ level: usageLevelLabel(routingSubTabMinLevel('singboxRouter')) })}</span>
	{/if}
{/snippet}

<Modal {open} title={m.tunnels_referenced_title()} size="sm" {onclose}>
	{#if details}
		<p class="lead">
			{entityLabel ?? m.tunnels_referenced_entity_default()} {#if tunnelName}<strong>{tunnelName}</strong>{/if} {m.tunnels_referenced_used_name_after()}
		</p>
		<ul class="ref-list">
			{#if details.deviceProxy}
				<li>{m.tunnels_referenced_device_proxy()}</li>
			{/if}
			{#if details.routerRules && details.routerRules.length > 0}
				<li>
					{m.tunnels_referenced_router_rules()}
					<span class="rule-indices">
						{details.routerRules.map((i) => `#${i}`).join(', ')}
					</span>
					{@render openLink('tproxy')}
				</li>
			{/if}
			{#if details.routerOther}
				{#each details.routerOther as loc}
					{@const ref = describeRouterReference(loc)}
					<li>
						{#if ref.known}{ref.text}{:else}<code>{ref.text}</code>{/if}
						{@render openLink(ref.plane)}
					</li>
				{/each}
			{/if}
		</ul>
		<p class="hint">{m.tunnels_referenced_hint()}</p>
	{/if}
	{#snippet actions()}
		<Button variant="primary" size="md" onclick={onclose}>{m.tunnels_referenced_ok()}</Button>
	{/snippet}
</Modal>

<style>
	.lead {
		margin: 0 0 0.5rem;
	}
	.ref-list {
		margin: 0.5rem 0 0.75rem;
		padding-left: 1.25rem;
		list-style: disc;
	}
	.ref-list li {
		margin: 0.25rem 0;
	}
	.rule-indices {
		font-family: var(--font-mono);
		color: var(--color-text-muted);
	}
	.ref-list code {
		font-family: var(--font-mono);
		font-size: 0.8125rem;
		color: var(--color-text-muted);
	}
	.level-note {
		color: var(--color-text-muted);
		font-size: 0.8125rem;
	}
	.hint {
		color: var(--color-text-muted);
		font-size: 0.875rem;
		margin: 0;
	}
</style>
