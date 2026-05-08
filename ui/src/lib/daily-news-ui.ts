import { Marked } from 'marked';
import DOMPurify from 'dompurify';

const dailyNewsMarked = new Marked({ breaks: true, gfm: true });

export type DailyNewsNavItem = {
	href: string;
	label: string;
	icon: string;
};

export type DailyNewsDigestDTO = {
	id: string;
	status: 'pending' | 'running' | 'success' | 'failed' | string;
	title?: string;
	body_markdown?: string;
	candidate_count?: number;
	included_count?: number;
	used_subset?: boolean;
	local_date?: string;
	generated_at?: string;
	error_message?: string;
};

export type DailyNewsStateMessage = {
	tone: 'info' | 'error' | 'empty';
	title: string;
	message: string;
};

export function dailyNewsNavItem(): DailyNewsNavItem {
	return { href: '/daily-news', label: 'Daily News', icon: '🗞️' };
}

export function dailyNewsLoadingMessage(): string {
	return 'Loading Daily News…';
}

export function dailyNewsSubsetMessage(digest: Pick<DailyNewsDigestDTO, 'used_subset' | 'included_count' | 'candidate_count'>): string {
	if (!digest.used_subset || !digest.included_count || !digest.candidate_count) {
		return '';
	}
	return `This digest is based on ${digest.included_count} of ${digest.candidate_count} available articles.`;
}

export function dailyNewsStateMessage(digest: DailyNewsDigestDTO | null | undefined): DailyNewsStateMessage | null {
	if (!digest) return null;
	if (digest.status === 'pending') {
		return { tone: 'info', title: 'Daily News is queued', message: 'Your digest has been queued and will be generated shortly.' };
	}
	if (digest.status === 'running') {
		return { tone: 'info', title: 'Daily News is being generated', message: 'Your digest is being written now. This page will update when it is ready.' };
	}
	if (digest.status === 'failed') {
		return { tone: 'error', title: 'Daily News generation failed', message: digest.error_message || 'Please try again later.' };
	}
	if (digest.status === 'success' && digest.candidate_count === 0 && !digest.body_markdown) {
		return { tone: 'empty', title: 'No articles today', message: 'No new articles matched this digest window.' };
	}
	return null;
}

function neutralizeDangerousMarkdownLinks(markdown: string): string {
	return markdown.replace(/!?\[([^\]]*)\]\(([^)]+)\)/g, (match, label: string, url: string) => {
		const trimmed = url.trim().toLowerCase();
		if (match.startsWith('!')) return label;
		if (trimmed.startsWith('https://')) return match;
		return label;
	});
}

export function renderDailyNewsMarkdown(markdown: string | null | undefined): string {
	if (!markdown) return '';
	const html = dailyNewsMarked.parse(neutralizeDangerousMarkdownLinks(markdown)) as string;
	return DOMPurify.sanitize(html, {
		ALLOWED_TAGS: [
			'h1',
			'h2',
			'h3',
			'h4',
			'h5',
			'h6',
			'p',
			'em',
			'strong',
			'blockquote',
			'ul',
			'ol',
			'li',
			'table',
			'thead',
			'tbody',
			'tr',
			'th',
			'td',
			'code',
			'pre',
			'br',
			'a'
		],
		ALLOWED_ATTR: ['href', 'title', 'target', 'rel'],
		ALLOW_DATA_ATTR: false,
		FORBID_TAGS: ['img', 'svg', 'script', 'style', 'iframe'],
		ADD_ATTR: ['target'],
		ADD_URI_SAFE_ATTR: [],
		ALLOWED_URI_REGEXP: /^https:\/\//i
	}).replace(/<a\s/gi, '<a target="_blank" rel="noopener noreferrer" ');
}
