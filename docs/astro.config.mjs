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
      customCss: ['./src/styles/custom.css'],
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
