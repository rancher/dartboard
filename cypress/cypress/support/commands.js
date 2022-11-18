// logs into Rancher setting the session cookie
Cypress.Commands.add('login', () => {
    cy.request({
        method: 'POST',
        url: '/v3-public/localProviders/local?action=login',
        body: {"description": "UI session", "responseType": "cookie", "username": "admin", "password": "adminpassword"},
        // setting curl as user-agent avoids a CSRF token check
        headers: {'accept': 'application/json', 'user-agent': 'curl/7.79.1'}
    })
})

Cypress.Commands.add('downstreamClusters', (f) => {
    cy.task('listDir', "../config").then(files => {
        files.forEach((file) => {
            const groups = file.match(/(downstream.+)\.yaml/)
            if (groups != null) {
                f(groups[1], file)
            }
        })
    })
})
