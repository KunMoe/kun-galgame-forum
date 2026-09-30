import withNuxt from './.nuxt/eslint.config.mjs'

const v1ClientMessage =
  'Reach the forum API only through the typed client in shared/utils/api/client.ts'

const pageRouteMessage =
  "Keep URL state in useQueryState (app/composables/useQueryState.ts), and read the route through Nuxt's page-scoped useRoute. The global route moves the moment a link is clicked, and a list being left refetches page 1."

export default withNuxt(
  {
    ignores: ['shared/types/api/**']
  },
  {
    files: [
      'app/**/*.{ts,vue}',
      'server/**/*.{ts,vue}',
      'shared/**/*.{ts,vue}'
    ],
    ignores: [
      'shared/utils/api/client.ts',
      'shared/utils/api/chat.ts',
      'shared/types/api/**'
    ],
    rules: {
      'no-restricted-syntax': [
        'error',
        {
          selector: 'Literal[value=/^\\/(api\\/)?v1(\\/|$)/]',
          message: v1ClientMessage
        },
        {
          selector: 'TemplateElement[value.raw=/^\\/(api\\/)?v1(\\/|$)/]',
          message: v1ClientMessage
        }
      ]
    }
  },
  {
    files: ['app/**/*.{ts,vue}'],
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            { name: '@vueuse/router', message: pageRouteMessage },
            {
              name: 'vue-router',
              importNames: ['useRoute'],
              message: pageRouteMessage
            }
          ]
        }
      ]
    }
  },
  {
    rules: {
      'no-console': 'off',
      camelcase: 'off',
      'comma-spacing': 'off',
      '@typescript-eslint/no-unused-vars': 'off',
      'vue/multi-word-component-names': 'off',
      'vue/singleline-html-element-content-newline': 'off',
      'vue/html-self-closing': 'off',
      'vue/attributes-order': 'off',
      'vue/no-multiple-template-root': 'off',
      'vue/no-v-html': 'off',
      'import/order': 'off',
      'import/no-named-as-default-member': 'off',
      'arrow-parens': ['error', 'always'],
      'space-before-function-paren': 'off',
      'func-call-spacing': 'off',
      quotes: [
        'error',
        'single',
        { avoidEscape: true, allowTemplateLiterals: true }
      ]
    }
  }
)
