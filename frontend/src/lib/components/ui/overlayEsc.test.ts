import { afterEach, describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import Modal from './Modal.svelte';
import ChipMultiSelect from './ChipMultiSelect.svelte';
import Dropdown from './Dropdown.svelte';
import SideDrawer from './SideDrawer.svelte';
import { createRawSnippet } from 'svelte';
import SubscriptionMemberPicker from '../subscriptions/SubscriptionMemberPicker.svelte';

// F432: оверлей поверх модалки забирает Esc себе. Modal слушает Esc на
// window во всплытии, и без остановки одно нажатие закрывало и оверлей, и
// модалку под ним.
describe('Esc в оверлее внутри модалки', () => {
	afterEach(() => vi.unstubAllGlobals());

	it('пикер члена подписки закрывается сам и не закрывает модалку', async () => {
		const closeModal = vi.fn();
		const closePicker = vi.fn();
		render(Modal, { props: { open: true, title: 'Подписка', onclose: closeModal } });
		render(SubscriptionMemberPicker, {
			props: { members: [], activeMemberTag: '', onPick: async () => {}, onClose: closePicker },
		});
		await new Promise((r) => setTimeout(r, 0)); // пикер вешает слушатель через тик
		await tick();

		await fireEvent.keyDown(document.body, { key: 'Escape' });

		expect(closePicker).toHaveBeenCalledTimes(1);
		expect(closeModal).not.toHaveBeenCalled();
	});

	it('открытый ChipMultiSelect закрывается сам и не закрывает модалку', async () => {
		const closeModal = vi.fn();
		render(Modal, { props: { open: true, title: 'Правило', onclose: closeModal } });
		const chip = render(ChipMultiSelect, {
			props: { values: [], options: [{ value: 'a', label: 'a' }], onchange: () => {} },
		});
		await tick();
		await fireEvent.click(chip.container.querySelector('button')!);
		await tick();

		await fireEvent.keyDown(document.body, { key: 'Escape' });

		expect(closeModal).not.toHaveBeenCalled();
	});

	it('открытый Dropdown закрывается сам и не закрывает модалку', async () => {
		// jsdom без ResizeObserver; Dropdown следит им за размером панели.
		vi.stubGlobal('ResizeObserver', class {
			observe() {}
			unobserve() {}
			disconnect() {}
		});
		const closeModal = vi.fn();
		render(Modal, { props: { open: true, title: 'Туннель', onclose: closeModal } });
		const dd = render(Dropdown, { props: { options: [{ value: 'a', label: 'a' }], label: 'Выбор' } });
		await tick();
		await fireEvent.click(dd.container.querySelector('button')!);
		await tick();
		const panel = document.querySelector('.dropdown-panel');
		expect(panel).not.toBeNull();

		await fireEvent.keyDown(panel!, { key: 'Escape' });

		expect(closeModal).not.toHaveBeenCalled();
		expect(document.querySelector('.dropdown-panel')).toBeNull();
	});

	// SideDrawer слушает Esc на document и зарегистрирован раньше списка —
	// stopPropagation с того же document его не остановил бы.
	it('открытый ChipMultiSelect не закрывает SideDrawer под собой', async () => {
		const closeDrawer = vi.fn();
		render(SideDrawer, {
			props: { open: true, onClose: closeDrawer, children: createRawSnippet(() => ({ render: () => '<div></div>' })) },
		});
		const chip = render(ChipMultiSelect, {
			props: { values: [], options: [{ value: 'a', label: 'a' }], onchange: () => {} },
		});
		await tick();
		await fireEvent.click(chip.container.querySelector('button')!);
		await tick();

		await fireEvent.keyDown(document.body, { key: 'Escape' });

		expect(closeDrawer).not.toHaveBeenCalled();
	});

	it('открытый Dropdown не закрывает SideDrawer под собой', async () => {
		vi.stubGlobal('ResizeObserver', class {
			observe() {}
			unobserve() {}
			disconnect() {}
		});
		const closeDrawer = vi.fn();
		render(SideDrawer, {
			props: { open: true, onClose: closeDrawer, children: createRawSnippet(() => ({ render: () => '<div></div>' })) },
		});
		const dd = render(Dropdown, { props: { options: [{ value: 'a', label: 'a' }], label: 'Выбор' } });
		await tick();
		await fireEvent.click(dd.container.querySelector('button')!);
		await tick();
		const panel = document.querySelector('.dropdown-panel');
		expect(panel).not.toBeNull();

		await fireEvent.keyDown(panel!, { key: 'Escape' });

		expect(closeDrawer).not.toHaveBeenCalled();
		expect(document.querySelector('.dropdown-panel')).toBeNull();
	});
});
