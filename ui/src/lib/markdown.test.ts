import { describe, it, expect } from 'vitest';
import { renderMarkdown, sanitizeHTML } from './markdown';

describe('renderMarkdown', () => {
	it('returns empty string for falsy input', () => {
		expect(renderMarkdown('')).toBe('');
		expect(renderMarkdown(null as unknown as string)).toBe('');
		expect(renderMarkdown(undefined as unknown as string)).toBe('');
	});

	it('converts markdown to HTML', () => {
		const result = renderMarkdown('**bold** text');
		expect(result).toContain('<strong>bold</strong>');
		expect(result).toContain('text');
	});

	it('converts links', () => {
		const result = renderMarkdown('[Go](https://go.dev)');
		expect(result).toContain('href="https://go.dev"');
		expect(result).toContain('Go');
	});

	it('converts code blocks', () => {
		const result = renderMarkdown('`code`');
		expect(result).toContain('<code>code</code>');
	});

	it('handles line breaks', () => {
		const result = renderMarkdown('line 1\nline 2');
		expect(result).toContain('<br>');
	});

	it('sanitizes dangerous HTML', () => {
		const result = renderMarkdown('<script>alert("xss")</script>');
		expect(result).not.toContain('<script>');
	});

	it('handles headers', () => {
		const result = renderMarkdown('# Title');
		expect(result).toContain('<h1');
		expect(result).toContain('Title');
	});

	it('handles bullet lists', () => {
		const result = renderMarkdown('- item 1\n- item 2');
		expect(result).toContain('<li>');
		expect(result).toContain('item 1');
	});
});

describe('sanitizeHTML', () => {
	it('returns empty string for falsy input', () => {
		expect(sanitizeHTML('')).toBe('');
		expect(sanitizeHTML(null as unknown as string)).toBe('');
	});

	it('passes safe HTML through', () => {
		const html = '<p>Hello <strong>world</strong></p>';
		expect(sanitizeHTML(html)).toContain('<p>');
		expect(sanitizeHTML(html)).toContain('<strong>');
	});

	it('removes script tags', () => {
		const html = '<p>Safe</p><script>alert("xss")</script>';
		const result = sanitizeHTML(html);
		expect(result).toContain('Safe');
		expect(result).not.toContain('<script>');
	});

	it('removes event handlers', () => {
		const html = '<a onclick="alert(1)">Link</a>';
		const result = sanitizeHTML(html);
		expect(result).not.toContain('onclick');
	});

	it('preserves links', () => {
		const html = '<a href="https://example.com">Link</a>';
		const result = sanitizeHTML(html);
		expect(result).toContain('href');
	});
});
