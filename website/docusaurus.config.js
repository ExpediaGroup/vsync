module.exports = {
  title: 'vsync',
  tagline: 'Sync secrets between HashiCorp vaults',
  url: 'https://expediagroup.github.io',
  baseUrl: '/vsync/',
  favicon: 'img/favicon.ico',
  organizationName: 'ExpediaGroup', // Usually your GitHub org/user name.
  projectName: 'vsync', // Usually your repo name.
  themeConfig: {
    navbar: {
      title: 'Vsync',
      logo: {
        alt: 'vsync logo',
        src: 'img/logo.svg',
      },
      links: [
        {to: 'docs/getstarted/why', label: 'Docs', position: 'left'},
        {
          href: 'https://github.com/ExpediaGroup/vsync',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      copyright: `Copyright © ${new Date().getFullYear()} Expedia, Inc. Built with Docusaurus.`,
    },
  },
  presets: [
    [
      '@docusaurus/preset-classic',
      {
        docs: {
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl:
            'https://github.com/ExpediaGroup/vsync/edit/master/website/',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
          algolia: {
            apiKey: 'bc2a3a8a5df178c4e174b77b084f0739',
            indexName: 'vsync',
            algoliaOptions: {}, // Optional, if provided by Algolia
          },
        },
      },
    ],
  ],
};
