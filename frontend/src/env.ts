import { defineEnvVars } from '@sveltejs/kit/env';

// Read when the server starts (not inlined at build time), matching the old
// $env/dynamic/public behaviour. Unset becomes '', and src/lib/config.ts falls
// back to http://localhost:3000 — the backend in the same Docker container.
export const variables = defineEnvVars({
	PUBLIC_BACKEND_URL: { public: true, schema: (input) => input ?? '' }
});
