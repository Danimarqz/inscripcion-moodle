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
