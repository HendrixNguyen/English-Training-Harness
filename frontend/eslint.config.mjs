import withNuxt from './.nuxt/eslint.config.mjs'

export default withNuxt({
  ignores: ['.output/**', '.nuxt/**', 'dist/**', 'dev-dist/**', 'public/**'],
})
