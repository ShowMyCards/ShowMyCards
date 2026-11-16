import { vi } from 'vitest';

// `$app/env/public` reads its values from the payload the SvelteKit runtime
// injects at SSR/hydration. Component tests run under vitest's browser mode,
// where that runtime doesn't exist, so importing the real module throws
// (`Cannot destructure property 'env' of 'globalThis.__sveltekit_dev'`).
//
// Stubbing it here, via the `client` project's `setupFiles`, lets any test
// that pulls in the `#lib` barrel — which re-exports `config.ts`, and
// `config.ts` reads this env — import cleanly. Variables are declared in
// src/env.ts; an unset one is '' at runtime, so stub it the same way.
vi.mock('$app/env/public', () => ({ PUBLIC_BACKEND_URL: '' }));
