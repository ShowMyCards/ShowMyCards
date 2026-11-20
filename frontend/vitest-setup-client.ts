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

// In browser mode, vi.mock only queues the mock: registering it is an async
// call to the Vitest server, and the runner imports the test file right after
// this setup file without waiting for it. A test file with its own vi.mock
// waits for the queue before importing, but one without any imports at once,
// and when its imports are fast (warm cache) it can reach the real module
// before the mock exists.
// Vitest routes dynamic imports through the mock queue, so awaiting one here
// holds the setup file until the mock is registered.
await import('$app/env/public');
