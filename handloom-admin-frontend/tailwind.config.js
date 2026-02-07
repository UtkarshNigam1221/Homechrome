/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#fef7f0',
          100: '#fdecd9',
          200: '#fad6b2',
          300: '#f6b880',
          400: '#f1924d',
          500: '#ec7428',
          600: '#dd5a1e',
          700: '#b7441a',
          800: '#92371d',
          900: '#76301b',
          950: '#40160c',
        },
        secondary: {
          50: '#f6f5f4',
          100: '#e8e6e3',
          200: '#d4cfca',
          300: '#b8b1a8',
          400: '#9a9085',
          500: '#857a6d',
          600: '#726659',
          700: '#5d544a',
          800: '#504841',
          900: '#453f39',
          950: '#25211e',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
