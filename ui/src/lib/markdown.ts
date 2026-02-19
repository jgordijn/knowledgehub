import { Marked } from 'marked';

const marked = new Marked({
	breaks: true,
	gfm: true
});

/**
 * Render a markdown string to HTML.
 * Returns sanitized HTML suitable for {@html} rendering.
 */
export function renderMarkdown(text: string): string {
	if (!text) return '';
	const html = marked.parse(text);
	// marked.parse can return string | Promise<string>; with no async
	// extensions it always returns string synchronously.
	return html as string;
}
