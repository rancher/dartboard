/// <reference types="cypress" />

describe('Rancher cluster import functionality', () => {
  beforeEach(function () {
    cy.login()
  })

  it('deletes any previously created clusters', () => {
    cy.visit("/")
    cy.waitTableLoaded()
    cy.get('.menu-icon').click()
    cy.contains('Cluster Management').click()
    cy.waitTableLoaded()

    cy.get("table").then($table => {
      if ($table.text().includes('test-cluster')) {
        cy.get("tr:contains('test-cluster') [role=checkbox]").click()
        cy.contains("Delete").click()
        cy.get("[role=dialog] button:contains('Delete')").click()

        cy.contains('test-cluster').should('not.exist')

        // HACK: deleting a cluster results in background operations lasting several seconds
        // if a new cluster is created in the meantime, creation will fail
        cy.wait(60000)
      }})
  })

  it('creates an imported cluster', () => {
    // HACK: not going back to the home page results in a Javascript error
    cy.visit("/")
    cy.waitTableLoaded()
    cy.get('.menu-icon').click()
    cy.contains('Cluster Management').click()
    cy.waitTableLoaded()

    cy.contains('Import Existing').click()
    cy.contains('Generic').click()
    cy.contains('label', 'Cluster Name').next('input').type('test-cluster')
    cy.contains('Create').click()
    
    cy.contains('curl --insecure').should('exist')
  })

  it('imports a cluster', () => {
    cy.contains("curl --insecure").then($e => {
      const registration_command = $e.text()
      cy.exec(`../config/ssh-*-downstream-server-node-0.sh '${registration_command}'`).then((result) => {
        cy.log(result.stdout)
      })
    })

    cy.visit("/")
    cy.waitTableLoaded()
    cy.get("tr:contains('test-cluster') .badge-state:contains('Active')", {timeout: 30*60*1000}).should("be.visible")
    cy.get("tr:contains('test-cluster') .badge-state:contains('Unavailable')", {timeout: 30*60*1000}).should("not.exist")
    cy.get("tr:contains('test-cluster') .badge-state:contains('Waiting')", {timeout: 30*60*1000}).should("not.exist")
    cy.get("tr:contains('test-cluster') .col-pods-usage > .icon", {timeout: 30*60*1000}).should("not.exist")
  })
})
