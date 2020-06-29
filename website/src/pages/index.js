import React from 'react';
import classnames from 'classnames';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import useBaseUrl from '@docusaurus/useBaseUrl';
import styles from './styles.module.css';

const features = [
  {
    title: <>Faster</>,
    // imageUrl: 'img/undraw_docusaurus_mountain.svg',
    description: (
      <>
        Parallel workers to finish the job faster
      </>
    ),
  },
  {
    title: <>Resilient</>,
    // imageUrl: 'img/undraw_docusaurus_mountain.svg',
    description: (
      <>
        Does not fail on copying single bad secret<br></br>
        No need of cron jobs to trigger syncing
      </>
    ),
  },
  {
    title: <>Cleaner</>,
    // imageUrl: 'img/undraw_docusaurus_tree.svg',
    description: (
      <>
        Vault audit logs are cleaner as vsync uses only kv metadata for comparison<br></br>
        Vsync closes all routines in each cycle cleanly on timeout
      </>
    ),
  },
  {
    title: <>Transformers</>,
    // imageUrl: 'img/undraw_docusaurus_react.svg',
    description: (
      <>
        Migrate paths of secrets from one format to another format and keep in sync without impacting developers and apps
      </>
    ),
  },
];

function Feature({ title, description}) {
  // const imgUrl = useBaseUrl(imageUrl);
  return (
    <div className={classnames('col col--4', styles.feature)}>
      {/* {imgUrl && (
        <div className="text--center">
          <img className={styles.featureImage} src={imgUrl} alt={title} />
        </div>
      )} */}
      <h3>{title}</h3>
      <p>{description}</p>
    </div>
  );
}

function Home() {
  const context = useDocusaurusContext();
  const {siteConfig = {}} = context;
  return (
    <Layout
      title={`Docs for ${siteConfig.title}`}
      description="Sync secrets between HashiCorp vaults <head />">
      <header className={classnames('hero', styles.heroBanner)}>
        <div className="container">
         <img src="img/vsync_text_animation.gif"/>
          <h1 className="hero__title">{siteConfig.title}</h1>
          <p className="hero__subtitle">{siteConfig.tagline}</p>
          <div className={styles.buttons}>
            <Link
              className={classnames(
                'button button--outline button--secondary button--lg',
                styles.getStarted,
              )}
              to={useBaseUrl('docs/getstarted/why')}>
              Get Started
            </Link>
          </div>
        </div>
      </header>
      <main>
        {features && features.length && (
          <section className={styles.features}>
            <div className="container">
              <div className="row">
                {features.map((props, idx) => (
                  <Feature key={idx} {...props} />
                ))}
              </div>
            </div>
          </section>
        )}
      </main>
    </Layout>
  );
}

export default Home;
