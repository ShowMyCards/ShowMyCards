/**
 * Backend API configuration
 */

import { PUBLIC_BACKEND_URL } from '$app/env/public';

/**
 * Base URL for the backend API server
 * Defaults to localhost:3000 in development
 * In production, configure via PUBLIC_BACKEND_URL environment variable
 */
export const BACKEND_URL = PUBLIC_BACKEND_URL || 'http://localhost:3000';
