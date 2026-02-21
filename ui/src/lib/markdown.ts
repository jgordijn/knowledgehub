import { Marked } from 'marked';
import DOMPurify from 'dompurify';

const marked = new Marked({
	breaks: true,
	gfm: true
});

/**
 * Render a markdown string to sanitized HTML.
 * Returns HTML safe for {@html} rendering.
 */
export function renderMarkdown(text: string): string {
	if (!text) return '';
	const html = marked.parse(text);
	// marked.parse can return string | Promise<string>; with no async
	// extensions it always returns string synchronously.
	return DOMPurify.sanitize(html as string);
}

/**
 * Sanitize raw HTML for safe {@html} rendering.
 */
export function sanitizeHTML(html: string): string {
	if (!html) return '';
	return DOMPurify.sanitize(html);
}
