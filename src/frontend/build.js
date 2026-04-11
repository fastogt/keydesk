const esbuild = require('esbuild');
const isWatch = process.argv.includes('--watch');

const shared = {
  bundle: true,
  sourcemap: true,
  target: 'es2020',
  logLevel: 'info',
};

async function build() {
  const setupOptions = {
    ...shared,
    entryPoints: ['setup.ts'],
    outdir: '../install/public/js',
    format: 'esm',
  };

  const dashboardOptions = {
    ...shared,
    entryPoints: ['dashboard.ts'],
    outdir: '../install/public/js',
    format: 'esm',
  };

  const peopleOptions = {
    ...shared,
    entryPoints: ['people.ts'],
    outdir: '../install/public/js',
    format: 'esm',
  };

  const accountsOptions = {
    ...shared,
    entryPoints: ['accounts.ts'],
    outdir: '../install/public/js',
    format: 'esm',
  };

  const servicesOptions = {
    ...shared,
    entryPoints: ['services.ts'],
    outdir: '../install/public/js',
    format: 'esm',
  };

  const settingsOptions = {
    ...shared,
    entryPoints: ['settings.ts'],
    outdir: '../install/public/js',
    format: 'esm',
  };

  const allOptions = [setupOptions, dashboardOptions, peopleOptions, accountsOptions, servicesOptions, settingsOptions];

  if (isWatch) {
    const contexts = await Promise.all(allOptions.map(o => esbuild.context(o)));
    await Promise.all(contexts.map(c => c.watch()));
    console.log('Watching for changes...');
  } else {
    await Promise.all(allOptions.map(o => esbuild.build(o)));
  }
}

build().catch((err) => {
  console.error(err);
  process.exit(1);
});
