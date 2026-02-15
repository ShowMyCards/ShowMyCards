/**
 * Backend API configuration
 */

import { env } from '$env/dynamic/public';

/**
 * Base URL for the backend API server
 * Defaults to localhost:3000 in development
 * In production, configure via PUBLIC_BACKEND_URL environment variable
 */
export const BACKEND_URL = env.PUBLIC_BACKEND_URL || 'http://localhost:3000';
