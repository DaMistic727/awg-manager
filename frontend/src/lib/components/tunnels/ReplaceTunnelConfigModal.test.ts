import { describe, it, expect, vi, beforeEach } from 'vitest';
import pako from 'pako';
import { render, fireEvent, screen, waitFor } from '@testing-library/svelte';
import ReplaceTunnelConfigModal from './ReplaceTunnelConfigModal.svelte';

const replaceConfig = vi.fn();

vi.mock('$lib/api/client', () => ({
	api: {
		replaceConfig: (...a: unknown[]) => replaceConfig(...a)
	}
}));

const PROPS = {
	open: true,
	tunnelId: 'awg5',
	tunnelName: 'fx-tunnel-old',
	tunnelState: 'stopped',
	backendLabel: 'Kernel',
	ndmsName: 'OpkgTun5',
	onclose: () => {},
	onreplaced: () => {}
};

/**
 * Ключ Premium в том виде, в каком его распознаёт vpnLinkUnsupportedPortalReason:
 * zlib-сжатый JSON в base64url. Секрета здесь нет — только признак типа ключа.
 */
function premiumVpnKey(): string {
	const raw = pako.deflate(JSON.stringify({ service_type: 'amnezia-premium' }));
	return 'vpn://' + btoa(String.fromCharCode(...raw)).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

beforeEach(() => {
	vi.clearAllMocks();
	replaceConfig.mockResolvedValue({ id: 'awg5', warnings: [] });
});

describe('ReplaceTunnelConfigModal', () => {
	it('ключ подписки Amnezia Premium во вкладке ссылки отвергается', async () => {
		render(ReplaceTunnelConfigModal, { props: { ...PROPS } });

		await fireEvent.click(screen.getByRole('button', { name: /Вставить ссылку/ }));
		const field = screen.getByPlaceholderText(/Вставьте vpn:\/\//);
		await fireEvent.input(field, { target: { value: premiumVpnKey() } });

		await waitFor(() => expect(screen.getByText(/Amnezia Premium не поддерживаются/)).toBeTruthy(), {
			timeout: 2000
		});
		// Конфига из ключа не получилось — заменять нечего.
		expect(screen.getByRole('button', { name: 'Заменить' }).hasAttribute('disabled')).toBe(true);
	});

	it('неизменённое длинное имя не показывает ошибку 256 байт', async () => {
		const longName = 'ф'.repeat(200); // 2 байта UTF-8 на символ → 400 байт
		render(ReplaceTunnelConfigModal, { props: { ...PROPS, tunnelName: longName } });

		expect(screen.queryByText(/длиннее 256 байт/)).toBeNull();
	});

	it('изменённое длинное имя показывает ошибку 256 байт', async () => {
		const longName = 'ф'.repeat(200);
		render(ReplaceTunnelConfigModal, { props: { ...PROPS } });

		const field = screen.getByLabelText('Имя туннеля');
		await fireEvent.input(field, { target: { value: longName } });

		expect(screen.getByText(/длиннее 256 байт/)).toBeTruthy();
	});
});
