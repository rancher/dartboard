/// <reference types="cypress" />

describe('Rancher cluster import functionality', () => {
  beforeEach(function () {
    cy.login()
  })

  it('deletes any previously created clusters', () => {
    cy.downstreamClusters(name => {
      cy.visit("/")
      cy.waitTableLoaded()
      cy.get('.menu-icon').click()
      cy.contains('Cluster Management').click()
      cy.waitTableLoaded()

      cy.get("table").then($table => {
        if ($table.text().includes(name)) {
          cy.get(`tr:contains('${name}') [role=checkbox]`).click()
          cy.contains("Delete").click()
          cy.get("[role=dialog] button:contains('Delete')").click()

          cy.contains('${cluster.name}').should('not.exist')

          // HACK: deleting a cluster results in background operations lasting several seconds
          // if a new cluster is created in the meantime, creation will fail
          cy.wait(60000)
        }
      })
    })
  })

  it('imports clusters', () => {
    cy.downstreamClusters((name, kubeconfig) => {
      // HACK: not going back to the home page results in a Javascript error
      cy.visit("/")
      cy.waitTableLoaded()
      cy.get('.menu-icon').click()
      cy.contains('Cluster Management').click()
      cy.waitTableLoaded()

      cy.contains('Import Existing').click()
      cy.contains('Generic').click()
      cy.contains('label', 'Cluster Name').next('input').type(name)
      cy.contains('Create').click()

      cy.contains('curl --insecure').should('exist').then($e => {
        const registration_command = $e.text()
        const groups = registration_command.match(/(\/v3\/import\/.+\.yaml)/)
        const registration_path = groups[1]
        cy.exec(`curl --insecure -sfL ${Cypress.config().baseUrl}/${registration_path} | kubectl --kubeconfig=../config/${kubeconfig} apply -f -`).then((result) => {
          cy.log(result.stdout)

          cy.visit("/")
          cy.waitTableLoaded()
          cy.get(`tr:contains('${name}') .badge-state:contains('Active')`, {timeout: 30 * 60 * 1000}).should("be.visible")
          cy.get(`tr:contains('${name}') .badge-state:contains('Unavailable')`, {timeout: 30 * 60 * 1000}).should("not.exist")
          cy.get(`tr:contains('${name}') .badge-state:contains('Waiting')`, {timeout: 30 * 60 * 1000}).should("not.exist")
          cy.get(`tr:contains('${name}') .col-pods-usage > .icon`, {timeout: 30 * 60 * 1000}).should("not.exist")
        })
      })
    })
  })
})
