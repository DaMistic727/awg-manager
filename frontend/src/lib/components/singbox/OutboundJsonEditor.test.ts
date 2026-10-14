import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import OutboundJsonEditor from './OutboundJsonEditor.svelte';

function setup() {
	const onchange = vi.fn();
	const oninvalid = vi.fn();
	render(OutboundJsonEditor, {
		props: { value: { type: 'vless', tag: 'proxy-01', server: 'a.example' }, onchange, oninvalid },
	});
	const ta = screen.getByRole<HTMLTextAreaElement>('textbox', { name: 'JSON outbound' });
	return { ta, onchange, oninvalid };
}

async function type(ta: HTMLTextAreaElement, text: string) {
	ta.value = text;
	await fireEvent.input(ta);
	await tick();
}

describe('OutboundJsonEditor', () => {
	it('показывает outbound без tag в поле с доступным именем', () => {
		const { ta } = setup();
		expect(JSON.parse(ta.value)).toEqual({ type: 'vless', server: 'a.example' });
	});

	it('валидный объект уходит в onchange', async () => {
		const { ta, onchange, oninvalid } = setup();
		await type(ta, '{"type":"vless","multiplex":{"enabled":true}}');
		expect(onchange).toHaveBeenLastCalledWith({ type: 'vless', multiplex: { enabled: true } });
		expect(oninvalid).toHaveBeenLastCalledWith(false);
	});

	it('битый JSON и не-объект — oninvalid(true) без onchange', async () => {
		const { ta, onchange, oninvalid } = setup();
		onchange.mockClear();
		for (const bad of ['{"type":', '[1]', 'null']) {
			oninvalid.mockClear();
			await type(ta, bad);
			expect(oninvalid).toHaveBeenCalledTimes(1);
			expect(oninvalid).toHaveBeenCalledWith(true);
		}
		expect(onchange).not.toHaveBeenCalled();
	});
});
