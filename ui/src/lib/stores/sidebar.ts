import { writable } from 'svelte/store';

export interface TierCount {
	label: string;
	id: string;       // HTML element id for scrolling
	count: number;
	icon: string;
}

export interface SidebarData {
	unreadCount: number;
	bookmarkedCount: number;
	resources: { id: string; name: string }[];
	selectedSources: Set<string>;
	sourceCounts: Map<string, number>;
	tierCounts: TierCount[];
	readFilter: 'unread' | 'all' | 'bookmarked';
	onToggleSource: (id: string) => void;
	onClearSources: () => void;
	onSetReadFilter: (filter: 'unread' | 'all' | 'bookmarked') => void;
}

const defaultData: SidebarData = {
	unreadCount: 0,
	bookmarkedCount: 0,
	resources: [],
	selectedSources: new Set(),
	sourceCounts: new Map(),
	tierCounts: [],
	readFilter: 'unread',
	onToggleSource: () => {},
	onClearSources: () => {},
	onSetReadFilter: () => {}
};

export const sidebarData = writable<SidebarData>(defaultData);
