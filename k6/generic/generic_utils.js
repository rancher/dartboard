import {check, sleep} from 'k6';
import http from 'k6/http';
import { jUnit, textSummary } from 'https://jslib.k6.io/k6-summary/0.1.0/index.js';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/refs/tags/3.0.1/dist/bundle.js";

// Required params: baseurl, coookies, data, cluster, namespace, iter
export function createConfigMaps(baseUrl, cookies, data, clusterId, namespace, iter) {
    const name = `test-config-map-${iter}`

    const url = clusterId === "local"?
      `${baseUrl}/v1/configmaps` :
      `${baseUrl}/k8s/clusters/${clusterId}/v1/configmaps`

    const res = http.post(`${url}`,
        JSON.stringify({
            "metadata": {
                "name": name,
                "namespace": namespace
            },
            "data": {"data": data}
        }),
        { cookies: cookies }
    )

    sleep(0.1)
    if (res.status != 201) {
        console.log(res)
    }
    check(res, {
        '/v1/configmaps returns status 201': (r) => r.status === 201,
    })
}


// Required params: baseurl, coookies, data, cluster, namespace, iter
export function createSecrets(baseUrl, cookies, data, clusterId, namespace, iter)  {
    const name = `test-secrets-${iter}`

    const url = clusterId === "local"?
      `${baseUrl}/v1/secrets` :
      `${baseUrl}/k8s/clusters/${clusterId}/v1/secrets`

    const res = http.post(`${url}`,
        JSON.stringify({
            "metadata": {
                "name": name,
                "namespace": namespace
            },
            "data": {"data": data},
            "type": "opaque"
        }),
        { cookies: cookies }
    )

    sleep(0.1)
    if (res.status != 201) {
        console.log(res)
    }
    check(res, {
        '/v1/secrets returns status 201': (r) => r.status === 201,
    })
}


// Required params: baseurl, coookies, cluster, namespace, iter
export function createDeployments(baseUrl, cookies, clusterId, namespace, iter) {
    const name = `test-deployment-${iter}`

    const url = clusterId === "local"?
      `${baseUrl}/v1/apps.deployments` :
      `${baseUrl}/k8s/clusters/${clusterId}/v1/apps.deployments`


    const res = http.post(`${url}`,
        JSON.stringify({
            "apiVersion": "apps/v1",
            "kind": "Deployment",
            "metadata": {
              "name": name,
              "namespace": namespace
            },
            "spec": {
              "selector": {
                "matchLabels": {
                  "app": name
                }
              },
              "template": {
                "metadata": {
                  "labels": {
                    "app": name
                  }
                },
                "spec": {
                  "containers": [
                    {
                      "command": ["sleep", "3600"],
                      "image": "busybox:latest",
                      "name": name
                    }
                  ],
                  "securityContext": {
                    "runAsUser": 2000,
                    "runAsGroup": 3000
                  }
                }
              }
            }
        }),
        { cookies: cookies }
    )

    sleep(0.1)
    if (res.status != 201) {
        console.log(res)
    }
    check(res, {
        '/v1/apps.deployments returns status 201': (r) => r.status === 201,
    })

}

/**
 * Generates a specialized JUnit XML report from k6 summary data, focusing on thresholds.
 *
 * @param {object} data - The k6 summary data object.
 * @returns {string} A string containing the JUnit XML report.
 */
function generateCustomJUnit(data) {
    let testcases = [];
    let failures = 0;

    const escapeXML = (str) => {
        if (typeof str !== 'string') return str;
        return str.replace(/[<>&'"]/g, (c) => {
            switch (c) {
                case '<': return '&lt;';
                case '>': return '&gt;';
                case '&': return '&amp;';
                case '\'': return '&apos;';
                case '"': return '&quot;';
            }
        });
    };

    for (const [metricName, metric] of Object.entries(data.metrics)) {
      if (!metric.thresholds) {
        continue;
      }

      for (const thresholdName in metric.thresholds) {
        const threshold = metric.thresholds[thresholdName];
        const isOk = threshold.ok;
        const testCaseName = `${metricName} - ${thresholdName}`;

        let testcase = `<testcase name="${escapeXML(testCaseName)}">`;

        if (!isOk) {
          failures++;
          const metricType = metric.type;
          let evidence = "No value available";

          // Provide specific evidence based on metric type
          if (metricType === 'trend') {
              // For trends, the threshold is on a specific percentile (e.g., "p(95)<200")
              const percentile = thresholdName.match(/p\(\s*(\d+\.?\d*)\s*\)/);
              if (percentile) {
                  const p = `p(${percentile[1]})`;
                  if (metric.values[p] !== undefined) {
                      evidence = `value for ${p} was ${metric.values[p]}`;
                  }
              }
          } else if (metricType === 'rate') {
              evidence = `rate was ${metric.values.rate * 100}%`;
          } else if (metricType === 'counter') {
              evidence = `count was ${metric.values.count}`;
          }

          const failureMessage = `Threshold not met. Evidence: ${evidence}.`;
          testcase += `<failure message="${escapeXML(failureMessage)}">`;
          testcase += `Description: The metric '${metricName}' failed the threshold '${thresholdName}'.\n`;
          testcase += `Evidence: ${evidence}.`;
          testcase += `</failure>`;
        }

        testcase += `</testcase>`;
        testcases.push(testcase);
      }
    }

    let xml = `<?xml version="1.0" encoding="UTF-8"?>\n`;
    xml += `<testsuites tests="${testcases.length}" failures="${failures}">\n`;
    xml += `  <testsuite name="k6-thresholds" tests="${testcases.length}" failures="${failures}">\n`;
    xml += `    ${testcases.join('\n    ')}\n`;
    xml += `  </testsuite>\n</testsuites>`;
    return xml;
}

/**
 * handleSummary processes the test results at the end of the k6 run
 * and generates multiple report formats.
 *
 * To use this, export it from your main k6 script:
 * `export { handleSummary } from './generic_utils.js';`
 *
 * See: https://grafana.com/docs/k6/latest/results-output/end-of-test/custom-summary/#use-handlesummary
 *
 * @param {object} data - The k6 summary data object.
 * @returns {object} An object where keys are file paths and values are the report content.
 */
export function handleSummary(data) {
  console.log('Finished executing test! Generating summary reports...');

  // 1. Write the color-enabled summary of the executed test to stdout.
  const stdout = textSummary(data, { indent: ' ', enableColors: true });

  return {
    'stdout': stdout,
    // 2. Write an xml JUnit results file.
    'junit.xml': jUnit(data),
    // 3. Write a JSON file with the summary of requests and metrics.
    'summary.json': JSON.stringify(data),
    // 4. Write an HTML file with the summary of request and metrics.
    'summary.html': htmlReport(data),
    // 5. Write a custom JUnit XML file with detailed threshold results.
    'junit-custom.xml': generateCustomJUnit(data),
  };
}
