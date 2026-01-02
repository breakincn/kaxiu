/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#FF6B35',
        'primary-dark': '#E85A2B',
        'primary-light': '#FFF4F0',
        secondary: '#4A90D9',
        'secondary-light': '#E8F4FD',
      }
    },
  },
  plugins: [],
}
