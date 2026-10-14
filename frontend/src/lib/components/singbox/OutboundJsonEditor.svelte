<!--
  Сырой JSON outbound для опций, у которых нет поля в форме (#683).
  tag в текст не попадает: он правится отдельным полем формы. Текст
  инициализируется один раз при монтировании; каждое валидное изменение
  отдаётся наверх через onchange, невалидное — через oninvalid.
-->
<script lang="ts">
	import { SyntaxHighlightedTextarea } from '$lib/components/ui';
	import { highlightJson } from '$lib/utils/shareEditorHighlight';
	import { parseJsonErrorPosition, type JsonErrorPos } from '$lib/utils/jsonErrorPosition';
	import { m } from '$lib/i18n';
	import { untrack } from 'svelte';

	type JsonRecord = Record<string, unknown>;

	interface Props {
		value: JsonRecord;
		onchange: (next: JsonRecord) => void;
		oninvalid: (invalid: boolean) => void;
	}

	let { value, onchange, oninvalid }: Props = $props();

	function initialText(): string {
		const { tag: _tag, ...rest } = value;
		return JSON.stringify(rest, null, 2);
	}

	let text = $state(initialText());
	let parseError = $state<{ message: string; pos: JsonErrorPos | null } | null>(null);

	// Эффект зависит только от text: колбэки читают состояние родителя
	// (outbound), и без untrack запись в него перезапускала бы эффект.
	$effect(() => {
		const current = text;
		untrack(() => report(current));
	});

	function report(current: string): void {
		let parsed: unknown;
		try {
			parsed = JSON.parse(current);
		} catch (e) {
			const message = e instanceof Error ? e.message : String(e);
			parseError = { message, pos: parseJsonErrorPosition(message, current) };
			oninvalid(true);
			return;
		}
		if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
			parseError = { message: m.singbox_edit_json_not_object(), pos: null };
			oninvalid(true);
			return;
		}
		parseError = null;
		oninvalid(false);
		onchange(parsed as JsonRecord);
	}
</script>

<div class="json-editor">
	<div class="editor-frame">
		<SyntaxHighlightedTextarea
			bind:value={text}
			ariaLabel={m.singbox_edit_section_json()}
			highlight={highlightJson}
			indentMode="json"
			wrap="pre"
			placeholder={'{ }'}
		/>
	</div>

	{#if parseError}
		<div class="feedback-error">
			<div class="feedback-title">
				{#if parseError.pos}
					{m.singbox_routing_slot_invalid_json_at({ line: parseError.pos.line, column: parseError.pos.column })}
				{:else}
					{m.singbox_routing_slot_invalid_json()}
				{/if}
			</div>
			<div class="feedback-body">{parseError.message}</div>
		</div>
	{:else}
		<p class="hint">{m.singbox_edit_json_hint()}</p>
	{/if}
</div>

<style>
	.json-editor {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	/* Моноширинные метрики на рамке наследуют оба слоя подсветки — так же,
	   как в ConfigSlotEditor. */
	.editor-frame {
		height: 28rem;
		padding: 0.5rem 0.6rem;
		background: var(--color-bg-primary);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
		font-size: 0.8rem;
		line-height: 1.45;
	}
	.editor-frame:focus-within {
		border-color: var(--color-primary, #3b82f6);
	}
	.editor-frame :global(.shl-stack) {
		height: 100%;
	}

	.hint {
		margin: 0;
		font-size: 0.8rem;
		line-height: 1.4;
		color: var(--color-text-secondary);
	}

	.feedback-error {
		padding: 0.5rem 0.65rem;
		border: 1px solid var(--color-error);
		border-radius: var(--radius-sm);
		background: color-mix(in srgb, var(--color-error) 10%, transparent);
		font-size: 0.82rem;
		line-height: 1.4;
	}
	.feedback-title {
		font-weight: 600;
		margin-bottom: 0.25rem;
	}
	.feedback-body {
		color: var(--color-text-secondary);
		word-break: break-word;
	}
</style>
