export type DailyNewsNavItem = {
	href: string;
	label: string;
	icon: string;
};

export function dailyNewsNavItem(): DailyNewsNavItem {
	return { href: '/daily-news', label: 'Daily News', icon: '🗞️' };
}

export function dailyNewsLoadingMessage(): string {
	return 'Loading Daily News…';
}
