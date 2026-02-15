import { type Icon as IconType } from '@lucide/svelte';

export type MenuItem = {
	name: string;
	href: string;
	icon: typeof IconType;
};
