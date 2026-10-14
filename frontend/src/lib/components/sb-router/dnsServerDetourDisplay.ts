import { m } from '$lib/i18n';
import type {
	SingboxProxyGroup,
	SingboxRouterDNSServer,
	SingboxRouterOutbound,
	SingboxTunnel,
	Subscription,
} from '$lib/types';
import type { OutboundGroup } from '$lib/components/routing/singboxRouter/outboundOptions';
import {
	normalizeDnsServerDetour,
} from '$lib/utils/dnsServerDetour';
import { resolveOutboundDisplay } from './adapters';
import type { OutboundDisplay } from './types';

/** Явный direct outbound в detour DNS-сервера. */
export function isDnsServerDirectDetour(detour?: string): boolean {
	return detour?.trim() === 'direct';
}

/** @deprecated use isDnsServerEmptyDetour from $lib/utils/dnsServerDetour */
export function isDnsServerViaRouteDetour(detour?: string): boolean {
	return normalizeDnsServerDetour(detour) === undefined;
}

/**
 * Подпись DNS-сервера для списков/дропдаунов: «<тип> · <адрес>».
 * У fakeip-типа нет upstream-адреса (он синтезирует адреса в туннель), поэтому
 * `server` пустой — показываем «fakeip · синтез», а не «fakeip · undefined».
 * Прочие типы без адреса (теоретически) сводятся к одному типу.
 */
export function dnsServerSubtitle(s: SingboxRouterDNSServer): string {
	const addr = s.server?.trim();
	if (addr) return `${s.type ?? 'dns'} · ${addr}`;
	if (s.type === 'fakeip') return m.sb_router_dns_fakeip_synth();
	return s.type ?? 'dns';
}

/**
 * DNS server detour chip:
 * - empty / direct (non-dns-direct) → «Напрямую»
 * - конкретный outbound → обычный мелкий бейдж цели
 */
export function dnsServerDetourDisplay(
	server: SingboxRouterDNSServer,
	outbounds: SingboxRouterOutbound[],
	outboundOptions: OutboundGroup[] = [],
	subscriptions: Subscription[] | null = null,
	proxyGroups: SingboxProxyGroup[] = [],
	singboxTunnels: SingboxTunnel[] = [],
): OutboundDisplay {
	const detour = server.detour?.trim() ?? '';

	if (detour === 'direct' || !detour) {
		return resolveOutboundDisplay(
			'direct',
			'direct',
			outbounds,
			outboundOptions,
			subscriptions,
			proxyGroups,
			singboxTunnels,
		);
	}

	return resolveOutboundDisplay(
		detour,
		'route',
		outbounds,
		outboundOptions,
		subscriptions,
		proxyGroups,
		singboxTunnels,
	);
}
