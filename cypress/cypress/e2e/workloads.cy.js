/// <reference types="cypress" />

const pod_count = Math.round((7 - 1) * 300 * 0.90)

describe('Rancher workload management', () => {
  beforeEach(function () {
    cy.login()
  })

  it('deletes any previously created deployments', () => {
    cy.visit("/")
    cy.waitTableLoaded()
    cy.contains('test-cluster').click()
    cy.waitTableLoaded()

    cy.contains('Workload').click()
    cy.waitTableLoaded()

    cy.contains('Deployments').click()
    cy.waitTableLoaded()

    cy.get("table").then($table => {
      if ($table.text().includes('nginx')) {
        cy.get("tr:contains('nginx') [role=checkbox]").click()
        cy.contains("Delete").click()
        cy.get("[role=dialog] button:contains('Delete')").click()

        cy.contains('nginx').should('not.exist', {timeout: 15*60_000})
      }})
  })

  it('adds a deployment to the cluster', () => {
    cy.contains('Create').click()
    cy.contains('Edit as YAML').click()

    cy.readFile('cypress/e2e/deployment.yaml').then(yaml =>{

      // simulate ctrl+a del
      cy.contains("apiVersion").click()
      cy.focused().type('{meta}a{del}')
      cy.get('.CodeMirror textarea').then($destination => {
        // simulate paste
        const pasteEvent = Object.assign(new Event("paste", { bubbles: true, cancelable: true }), {
          clipboardData: {
            getData: () => yaml
          }
        });
        $destination[0].dispatchEvent(pasteEvent);

        cy.contains('Create').click()

        cy.get(":contains('Redeploy'):visible")
        cy.contains('nginx').click()

        cy.get(`:contains('1 Running')`, {timeout: 15*60_000}).should("be.visible")
      });
    })
  })

  it('scales a deployment up', () => {
    cy.contains('Deployments').click()
    cy.waitTableLoaded()
    cy.get("tr:contains('nginx') .icon-actions").click()
    cy.contains('Edit Config').click()

    cy.contains('label', 'Replicas').next('input').type(`{backspace}{backspace}{backspace}{backspace}{del}{del}{del}{del}${pod_count}`)

    cy.contains('Save').click()
    cy.get(":contains('Redeploy'):visible")
    cy.contains('nginx', {timeout: 10_000}).click()

    cy.get(`:contains('${pod_count} Running')`, {timeout: 60*60_000}).should("be.visible")
  })

  it('drains a worker node', () => {
    cy.visit("/")
    cy.waitTableLoaded()
    cy.contains('test-cluster').click()
    cy.contains('Cluster Dashboard').should('be.visible')
    cy.waitTableLoaded()
    cy.contains('Nodes').click()
    cy.waitTableLoaded()

    cy.get("tr:contains('Worker') [role=checkbox]").each(($e, i) => {
      if (i === 0) {
        cy.wrap($e).click().then(() =>{
          if (Cypress.$("#drain:enabled").length > 0) {
            cy.contains('Drain').click()
            cy.get("[role=dialog] label:contains('Yes')").each($e =>{
              cy.wrap($e).click()
            })
            cy.get("[role=dialog] button:contains('Drain')").click()
            cy.get(".badge-state:contains('Drained')", {timeout: 60_000}).should("not.exist")
          }
        })
      }
    })
  })

  it('recycles the deployment', () => {
    cy.visit("/")
    cy.waitTableLoaded()
    cy.contains('test-cluster').click()
    cy.waitTableLoaded()

    cy.contains('Workload').click()
    cy.waitTableLoaded()

    cy.contains('Deployments').click()
    cy.waitTableLoaded()

    cy.get("button:contains('Redeploy')")
    cy.get("tr:contains('nginx') [role=checkbox]").click()
    cy.get("button:contains('Redeploy')").click()

    cy.contains('nginx').click()
    cy.get(`:contains('${pod_count} Running')`, {timeout: 60*60_000}).should("be.visible")
  })

  it('uncordons all nodes', () => {
    cy.contains('Cluster').click()
    cy.waitTableLoaded()
    cy.contains('Nodes').click()
    cy.waitTableLoaded()

    cy.get("thead tr [role=checkbox]").click()

    cy.get("button:contains('Uncordon'):enabled", {timeout: 60_000}).click()
    cy.get(".badge-state:contains('Drained')", {timeout: 60_000}).should("not.exist")
  })

  it('recycles the deployment', () => {
    cy.visit("/")
    cy.waitTableLoaded()
    cy.contains('test-cluster').click()
    cy.waitTableLoaded()

    cy.contains('Workload').click()
    cy.waitTableLoaded()

    cy.contains('Deployments').click()
    cy.waitTableLoaded()

    cy.get("button:contains('Redeploy')")
    cy.get("tr:contains('nginx') [role=checkbox]").click()
    cy.get("button:contains('Redeploy')").click()

    cy.contains('nginx').click()
    cy.get(`:contains('${pod_count} Running')`, {timeout: 60*60_000}).should("be.visible")
  })
})
