import { vi } from 'vitest';

// `$env/dynamic/public` is a virtual module that the SvelteKit runtime
// populates at SSR/hydration. Component tests run under vitest's browser
// mode, where that runtime doesn't exist, so importing the real module
// throws (`Cannot read properties of undefined (reading 'env')`).
//
// Stubbing it here, via the `client` project's `setupFiles`, lets any test
// that pulls in the `$lib` barrel — which re-exports `config.ts`, and
// `config.ts` reads this env — import cleanly.
vi.mock('$env/dynamic/public', () => ({ env: {} }));
