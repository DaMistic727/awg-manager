import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import CreateManagedServerModal from './CreateManagedServerModal.svelte';
import { api } from '$lib/api/client';
import { notifications } from '$lib/stores/notifications';

vi.mock('$lib/api/client', () => ({
	api: {
		createManagedServer: vi.fn(),
		getWANIP: vi.fn(() => Promise.resolve('')),
		suggestManagedServerAddress: vi.fn(() => Promise.resolve({ address: '10.0.0.1', mask: '24' })),
	},
}));
vi.mock('$lib/stores/notifications', () => ({
	notifications: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

function apiError(code: string, message: string): Error {
	return Object.assign(new Error(message), { status: 400, body: { success: false, code, message } });
}

async function submit(onclose = vi.fn(), onCreated = vi.fn()) {
	const { getByText } = render(CreateManagedServerModal, { open: true, onclose, onCreated });
	await fireEvent.click(getByText('Создать').closest('button') as HTMLButtonElement);
	return { onclose, onCreated };
}

describe('CreateManagedServerModal', () => {
	beforeEach(() => vi.clearAllMocks());

	// Сервер создан, не сверен только ASC: модалка закрывается как после
	// успеха, отказ показан предупреждением, а не ошибкой (F594).
	it('ASC_UNVERIFIED закрывает модалку как созданный сервер', async () => {
		vi.mocked(api.createManagedServer).mockRejectedValue(apiError('ASC_UNVERIFIED', 'ASC не сверен'));
		const { onclose, onCreated } = await submit();
		await waitFor(() => expect(onCreated).toHaveBeenCalledTimes(1));
		expect(onclose).toHaveBeenCalledTimes(1);
		expect(notifications.warning).toHaveBeenCalledWith('ASC не сверен');
		expect(notifications.error).not.toHaveBeenCalled();
	});

	it('прочий отказ оставляет модалку открытой', async () => {
		vi.mocked(api.createManagedServer).mockRejectedValue(apiError('CREATE_FAILED', 'нет'));
		const { onclose, onCreated } = await submit();
		await waitFor(() => expect(notifications.error).toHaveBeenCalledWith('нет'));
		expect(onclose).not.toHaveBeenCalled();
		expect(onCreated).not.toHaveBeenCalled();
	});
});
