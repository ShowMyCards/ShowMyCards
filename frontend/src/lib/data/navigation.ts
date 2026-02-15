import type { MenuItem } from '$lib/types/navigation';
import {
	LayoutDashboard,
	Cog,
	Search,
	Archive,
	Scale,
	ListTodo,
	Grid2x2,
	Activity,
	Upload
} from '@lucide/svelte';

export const menuItems: MenuItem[] = [
	{
		name: 'Dashboard',
		href: '/',
		icon: LayoutDashboard
	},
	{
		name: 'Search',
		href: '/search',
		icon: Search
	},
	{
		name: 'Import',
		href: '/import',
		icon: Upload
	},
	{
		name: 'Inventory',
		href: '/inventory',
		icon: Grid2x2
	},
	{
		name: 'Lists',
		href: '/lists',
		icon: ListTodo
	},
	{
		name: 'Storage Locations',
		href: '/storage',
		icon: Archive
	},
	{
		name: 'Storage Rules',
		href: '/rules',
		icon: Scale
	},
	{
		name: 'Jobs',
		href: '/jobs',
		icon: Activity
	},
	{
		name: 'Settings',
		href: '/settings',
		icon: Cog
	}
];
