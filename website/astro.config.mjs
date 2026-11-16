// @ts-check
import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
  output: 'static',

  // Astro 7 changed the default to 'jsx', which strips whitespace between
  // tags and runs adjacent words and links together. Keep Astro 6's behaviour.
  compressHTML: true,

  vite: {
    plugins: [tailwindcss()]
  }
});
