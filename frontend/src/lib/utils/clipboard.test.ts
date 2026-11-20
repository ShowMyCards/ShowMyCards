import { afterEach, describe, expect, it, vi } from 'vitest';
import { copyText } from './clipboard';

/** A stand-in for the preview textarea the legacy copy path selects. */
function fakeTextarea() {
	return { focus: vi.fn(), select: vi.fn() } as unknown as HTMLTextAreaElement;
}

function stubDocument(execCommand: () => boolean) {
	const exec = vi.fn(execCommand);
	vi.stubGlobal('document', { execCommand: exec });
	return exec;
}

describe('copyText', () => {
	afterEach(() => vi.unstubAllGlobals());

	it('uses the Clipboard API when it is available', async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		vi.stubGlobal('navigator', { clipboard: { writeText } });
		const exec = stubDocument(() => true);

		await expect(copyText('1 Lightning Bolt', fakeTextarea())).resolves.toBe('copied');
		expect(writeText).toHaveBeenCalledWith('1 Lightning Bolt');
		expect(exec).not.toHaveBeenCalled();
	});

	it('falls back to selecting the textarea when the Clipboard API is missing (plain HTTP)', async () => {
		vi.stubGlobal('navigator', {});
		const exec = stubDocument(() => true);
		const textarea = fakeTextarea();

		await expect(copyText('1 Lightning Bolt', textarea)).resolves.toBe('copied');
		expect(textarea.select).toHaveBeenCalled();
		expect(exec).toHaveBeenCalledWith('copy');
	});

	it('falls back when the Clipboard API rejects', async () => {
		vi.stubGlobal('navigator', {
			clipboard: { writeText: vi.fn().mockRejectedValue(new Error('denied')) }
		});
		stubDocument(() => true);

		await expect(copyText('1 Lightning Bolt', fakeTextarea())).resolves.toBe('copied');
	});

	it('asks for a manual copy when every method fails, leaving the text selected', async () => {
		vi.stubGlobal('navigator', {});
		stubDocument(() => false);
		const textarea = fakeTextarea();

		await expect(copyText('1 Lightning Bolt', textarea)).resolves.toBe('manual');
		expect(textarea.select).toHaveBeenCalled();
	});

	it('asks for a manual copy when there is no textarea to fall back to', async () => {
		vi.stubGlobal('navigator', {});
		stubDocument(() => true);

		await expect(copyText('1 Lightning Bolt')).resolves.toBe('manual');
	});
});
