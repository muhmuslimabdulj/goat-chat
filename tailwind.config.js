/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./view/**/*.templ",
    "./static/js/**/*.js",
  ],
  theme: {
    extend: {
      colors: {
        'brutal-bg': '#E0E0E0',
        'neon-yellow': '#FFD100',
        'neon-pink': '#FF6AC1',
        'neon-green': '#00E676',
        'neon-cyan': '#00E5FF',
        'neon-red': '#FF5252',
        'neon-purple': '#B388FF',
        'neon-orange': '#FF9100',
        'neon-mint': '#69F0AE',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
      boxShadow: {
        'brutal': '8px 8px 0 0 rgba(0, 0, 0, 1)',
        'brutal-sm': '4px 4px 0 0 rgba(0, 0, 0, 1)',
      },
    },
  },
  plugins: [],
}
