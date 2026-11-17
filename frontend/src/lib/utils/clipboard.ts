/** 'copied' when the text reached the clipboard; 'manual' when the user must press Ctrl/Cmd+C. */
export type CopyResult = 'copied' | 'manual';

/**
 * Copy text to the clipboard.
 *
 * The Clipboard API only exists in secure contexts (HTTPS or localhost), so a
 * self-hosted instance opened over plain HTTP on the LAN has none. In that case,
 * or when the API rejects, select `fallback` (a textarea showing the same text)
 * and use the legacy copy command. If that fails too, the text is left selected so
 * the user can copy it by hand.
 */
export async function copyText(text: string, fallback?: HTMLTextAreaElement): Promise<CopyResult> {
	try {
		if (navigator.clipboard?.writeText) {
			await navigator.clipboard.writeText(text);
			return 'copied';
		}
	} catch {
		// Permission denied or not focused: fall through to the legacy path.
	}

	if (!fallback) return 'manual';

	fallback.focus();
	fallback.select();
	try {
		// Deprecated, but the only copy mechanism outside secure contexts.
		return document.execCommand('copy') ? 'copied' : 'manual';
	} catch {
		return 'manual';
	}
}
