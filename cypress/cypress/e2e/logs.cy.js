/// <reference types="cypress" />

describe('Log collection', () => {
  it('collects relevant logs', () => {
    cy.exec(`cd ..; ./util/collect_logs.sh`, {timeout: 60 * 60_000}).then((result) => {
      cy.log(result.stdout)
    })
  })
})
