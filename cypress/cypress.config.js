const { defineConfig } = require("cypress");

module.exports = defineConfig({
  e2e: {
    baseUrl: 'https://upstream.local.gd:3000/',
    specPattern: [
      "cypress/e2e/users.*",
      "cypress/e2e/imported-clusters.*",
      "cypress/e2e/workloads.*",
      "cypress/e2e/rke2-update.*",
      "cypress/e2e/logs.*",
    ]
  },
  viewportWidth: 1920,
  viewportHeight: 1080
});
