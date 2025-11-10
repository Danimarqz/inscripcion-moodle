/** @type {import('tailwindcss').Config} */
export default {
	content: [
    './src/**/*.astro',
    './src/**/*.html',
    './src/**/*.jsx',
    './src/**/*.tsx',
    './src/**/*.md',
    './src/**/*.mdx',
    './public/**/*.html',
  ],
	theme: {
		extend: {
      colors: {
        // Brand palette (for specific UI elements)
        brand: {
          pink: '#fb1159',
          yellow: '#faa40b',
          blue: '#0f99bc',
        },
        'brand-pink': '#fb1159',
        'brand-yellow': '#faa40b',
        'brand-blue': '#0f99bc',
        // Original accent palette
        accent: '#883aea',
        'accent-light': '#e0ccfa',
        'accent-dark': '#310a65',
        dark: {
          100: '#13151a',
          200: '#1a1c22',
          300: '#23262d',
          400: '#2a2d33'
        }
      }
    },
	},
	plugins: [],
}
