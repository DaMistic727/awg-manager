import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import type { Settings, TunnelReferencedError } from '$lib/types';

const { routerSettings, reloadSettings } = vi.hoisted(() => {
	// vi.mock поднимается выше импортов — стор создаём здесь же, без svelte/store.
	let value: { routingMode?: string } | null = null;
	const subs = new Set<(v: typeof value) => void>();
	return {
		routerSettings: {
			set(v: typeof value) { value = v; subs.forEach((f) => f(v)); },
			subscribe(f: (v: typeof value) => void) { subs.add(f); f(value); return () => subs.delete(f); },
		},
		reloadSettings: vi.fn(async () => {}),
	};
});
vi.mock('$lib/stores/singboxRouter', () => ({
	singboxRouter: { settings: { subscribe: routerSettings.subscribe }, reloadSettings },
}));
vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

import TunnelReferencedModal from './TunnelReferencedModal.svelte';
import { setSettings } from '$lib/stores/settings';
import { usageLevelLabel } from '$lib/types/usageLevel';

// Правило TProxy и ссылка из FakeIP-слота — обе плоскости в одной модалке.
const details = {
	routerRules: [1],
	routerOther: ['[fakeip] route.rules[0]'],
} as unknown as TunnelReferencedError;

function show(level: Settings['usageLevel'], routingMode?: string) {
	setSettings({ usageLevel: level } as Settings);
	routerSettings.set(routingMode ? { routingMode } : null);
	render(TunnelReferencedModal, { props: { open: true, details, onclose: () => {} } });
	return tick();
}

// F640: вкладки sing-box видны с «Эксперта» (FakeIP — ещё в fakeip-tun);
// ссылка «Открыть» на невидимую вкладку — тупик, вместо неё подсказка уровня.
describe('TunnelReferencedModal: ссылки по уровню', () => {
	beforeEach(() => reloadSettings.mockClear());

	it('эксперт видит обе ссылки', async () => {
		await show('expert');
		expect(screen.queryAllByRole('link')).toHaveLength(2);
	});

	it('advanced без fakeip-tun — ни одной ссылки, подсказки уровня', async () => {
		await show('advanced');
		expect(screen.queryAllByRole('link')).toHaveLength(0);
		const notes = document.querySelectorAll('.level-note');
		expect(notes).toHaveLength(2);
		expect(notes[0].textContent).toContain(usageLevelLabel('expert'));
	});

	it('advanced в fakeip-tun — ссылка только на FakeIP', async () => {
		await show('advanced', 'fakeip-tun');
		const links = screen.queryAllByRole('link');
		expect(links).toHaveLength(1);
		expect(links[0].getAttribute('href')).toBe('/routing?tab=fakeip');
	});

	it('режим движка не загружен — подгружается при открытии', async () => {
		await show('advanced');
		expect(reloadSettings).toHaveBeenCalled();
	});

	it('режим движка известен — повторно не грузится', async () => {
		await show('advanced', 'tproxy');
		expect(reloadSettings).not.toHaveBeenCalled();
	});
});
