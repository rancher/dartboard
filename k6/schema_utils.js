import { check, fail, sleep } from 'k6';
import http, { del } from 'k6/http'

export const schemasTag = { url: `/v1/schemas/<schemaID>` }
export const schemaDefinitionTag = { url: `/v1/schemaDefinitions/<schemaID>` }

export function getSchema(baseUrl, cookies, existingID) {
  let res = http.get(
    `${baseUrl}/v1/schemas/${existingID}`,
    {
      cookies: cookies,
      tags: schemasTag,
    }
  )
  return res
}

export function verifySchemaExistsPolling(baseUrl, cookies, existingID, expectedVersion, timeoutMs) {
  const timeWas = new Date();
  let timeSpent = null
  let res = null
  let currentVersion = ""
  // Poll schemaDefinition until receiving a 200
  while (new Date() - timeWas < timeoutMs) {
    res = retryUntilExpected(200, () => { return getSchema(baseUrl, cookies, existingID) })
    timeSpent = new Date() - timeWas
    console.log("SCHEMA STATUS: ", res.status)
    if (res.status === 200) {
      currentVersion = JSON.parse(res.body).attributes.version
      if (currentVersion === expectedVersion) {
        console.log("Polling conditions met after ", timeSpent, "ms");
        break;
      }
    }
  }
  const criteria = {}
  criteria[`GET /v1/schemas/<schemaID> returns status 200`] = (r) => r.status === 200
  criteria[`detected the expected schema version "${expectedVersion}" matches the received version`] = (r) => currentVersion === expectedVersion
  check(res, criteria)
  return { res: res, timeSpent: timeSpent }
}

export function getSchemaDefinition(baseUrl, cookies, existingID) {
  let res = http.get(
    `${baseUrl}/v1/schemaDefinitions/${existingID}`,
    {
      cookies: cookies,
      tags: schemaDefinitionTag,
    }
  )
  return res
}

export function verifySchemaDefinitionExistsPolling(baseUrl, cookies, existingID, expectedVersion, timeoutMs) {
  const criteria = {}
  let definitionType = ""
  criteria[`GET /v1/schemaDefinitions/<schemaID> returns status 200`] = (r) => r.status === 200
  criteria[`verify the definitionType includes ${expectedVersion}`] = (r) => JSON.parse(r.body).definitionType.includes(expectedVersion)

  const timeWas = new Date();
  let timeSpent = null
  let res = null
  // Poll schemaDefinition until receiving a 200
  while (new Date() - timeWas < timeoutMs) {
    res = retryUntilExpected(200, () => { return getSchemaDefinition(baseUrl, cookies, existingID) })
    timeSpent = new Date() - timeWas
    console.log("SCHEMADEFINITION STATUS: ", res.status)
    if (res.status === 200) {
      definitionType = JSON.parse(res.body).definitionType
      if (definitionType.includes(expectedVersion)) {
        console.log("Polling conditions met after ", timeSpent, "ms");
        break;
      }
    }
  }
  console.log("FINISHED POLLING AFTER", new Date() - timeWas)
  console.log(`VERIFY DEFINITIONTYPE (${definitionType} includes ${expectedVersion}): `, definitionType.includes(expectedVersion))
  check(res, criteria)
  return { res: res, timeSpent: timeSpent }
}
