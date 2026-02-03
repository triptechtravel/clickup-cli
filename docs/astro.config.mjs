import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://triptechtravel.github.io',
  base: '/clickup-cli',
  legacy: {
    collections: true,
  },
  integrations: [
    starlight({
      title: 'clickup CLI',
      logo: {
        src: './src/assets/logo.svg',
        alt: 'clickup CLI',
        replacesTitle: false,
      },
      favicon: '/favicon.svg',
      customCss: ['./src/styles/custom.css'],
      head: [
        { tag: 'meta', attrs: { property: 'og:image', content: 'https://triptechtravel.github.io/clickup-cli/og-image.png' } },
        { tag: 'meta', attrs: { property: 'og:image:width', content: '1200' } },
        { tag: 'meta', attrs: { property: 'og:image:height', content: '630' } },
        { tag: 'meta', attrs: { property: 'og:description', content: 'A command-line tool for working with ClickUp tasks, comments, and sprints -- designed for developers who live in the terminal and use GitHub.' } },
        { tag: 'meta', attrs: { name: 'twitter:card', content: 'summary_large_image' } },
      ],
      social: {
        github: 'https://github.com/triptechtravel/clickup-cli',
      },
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Installation', slug: 'installation' },
            { label: 'Getting Started', slug: 'getting-started' },
          ],
        },
        {
          label: 'Usage',
          items: [
            { label: 'Commands', slug: 'commands' },
            { label: 'Configuration', slug: 'configuration' },
            { label: 'Git Integration', slug: 'git-integration' },
          ],
        },
        {
          label: 'Integrations',
          items: [
            { label: 'CI Usage', slug: 'ci-usage' },
            { label: 'GitHub Actions', slug: 'github-actions' },
            { label: 'AI Agents', slug: 'ai-agents' },
          ],
        },
        {
          label: 'Project',
          items: [
            { label: 'Contributing', slug: 'contributing' },
            { label: 'Security', slug: 'security' },
          ],
        },
      ],
    }),
  ],
});
