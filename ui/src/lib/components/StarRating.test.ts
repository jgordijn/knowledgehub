import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, unmount } from 'svelte';
import StarRating from './StarRating.svelte';

describe('StarRating', () => {
	let target: HTMLElement;

	beforeEach(() => {
		target = document.createElement('div');
		document.body.appendChild(target);
	});

	it('renders 5 star buttons', () => {
		const component = mount(StarRating, { target, props: { aiStars: 3 } });
		const buttons = target.querySelectorAll('button');
		expect(buttons.length).toBe(5);
		unmount(component);
	});

	it('renders correct aria labels', () => {
		const component = mount(StarRating, { target, props: { aiStars: 3 } });
		expect(target.querySelector('[aria-label="Rate 1 star"]')).toBeTruthy();
		expect(target.querySelector('[aria-label="Rate 2 stars"]')).toBeTruthy();
		expect(target.querySelector('[aria-label="Rate 5 stars"]')).toBeTruthy();
		unmount(component);
	});

	it('has star group for accessibility', () => {
		const component = mount(StarRating, { target, props: { aiStars: 3 } });
		expect(target.querySelector('[role="group"]')).toBeTruthy();
		unmount(component);
	});

	it('calls onRate when a star is clicked', async () => {
		const onRate = vi.fn();
		const component = mount(StarRating, { target, props: { aiStars: 3, onRate } });

		const star4 = target.querySelector('[aria-label="Rate 4 stars"]') as HTMLButtonElement;
		star4?.click();

		expect(onRate).toHaveBeenCalledWith(4);
		unmount(component);
	});

	it('renders with default values', () => {
		const component = mount(StarRating, { target });
		const buttons = target.querySelectorAll('button');
		expect(buttons.length).toBe(5);
		unmount(component);
	});
});
