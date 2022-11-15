const { defineConfig } = require("cypress");

const fs = require('fs')

module.exports = defineConfig({
  e2e: {
    baseUrl: 'https://upstream.local.gd:3000/',
    specPattern: [
      "cypress/e2e/users.*",
      "cypress/e2e/imported-clusters.*",
      // "cypress/e2e/workloads.*",
      // "cypress/e2e/rke2-update.*",
      // "cypress/e2e/logs.*",
    ],
    setupNodeEvents(on, config) {
      on('task', {
        listDir(path) {
          return fs.readdirSync(path)
        },
      })
    }
  },
  viewportWidth: 1920,
  viewportHeight: 1080
});
