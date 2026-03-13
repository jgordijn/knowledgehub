import { writable } from 'svelte/store';

export interface SidebarData {
	unreadCount: number;
	bookmarkedCount: number;
	resources: { id: string; name: string }[];
	selectedSources: Set<string>;
	sourceCounts: Map<string, number>;
	onToggleSource: (id: string) => void;
	onClearSources: () => void;
}

const defaultData: SidebarData = {
	unreadCount: 0,
	bookmarkedCount: 0,
	resources: [],
	selectedSources: new Set(),
	sourceCounts: new Map(),
	onToggleSource: () => {},
	onClearSources: () => {}
};

export const sidebarData = writable<SidebarData>(defaultData);
