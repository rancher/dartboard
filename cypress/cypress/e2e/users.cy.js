/// <reference types="cypress" />

describe('Rancher user management', () => {
  it('creates the first user', () => {
    cy.visit('/')
    cy.contains('Welcome').should("be.visible")

    if (Cypress.$(":contains('Howdy'):visible").length === 0) {
      cy.contains('Set a specific password to use').click()

      cy.contains('label', 'Bootstrap Password').next('input').type('admin')

      cy.contains('label', 'New Password').next('input').type('adminpassword')
      cy.contains('label', 'Confirm New Password').next('input').type('adminpassword')

      cy.get("label[for='checkbox-telemetry'] .checkbox-custom").click()
      cy.get("label[for='checkbox-eula'] .checkbox-custom").click()
      cy.contains('Continue').click()

      cy.contains('Getting Started', {timeout: 10_000}).should('exist')
    }
  })

  it('logs a user in', () => {
    cy.visit('/')

    cy.contains('label', 'Username').next('input').type('admin')
    cy.contains('label', 'Password').next('input').type('adminpassword')
    cy.contains('Log in with Local User').click()

    cy.contains('Getting Started').should('exist')
  })
})
