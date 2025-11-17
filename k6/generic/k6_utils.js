import { jUnit, textSummary } from '../lib/k6-summary-0.1.0.js';

import { htmlReport } from '../lib/k6-reporter-3.0.1.js';
import { getPathBasename } from '../generic/generic_utils.js';

export function sizeOfHeaders(headers) {
  return Object.keys(headers).reduce((sum, key) => sum + key.length + headers[key].length, 0);
}

export function trackResponseSizePerURL(res, tags, headerDataRecv, epDataRecv) {
  // Add data points for received data
  headerDataRecv.add(sizeOfHeaders(res.headers));
  if (res.hasOwnProperty('body') && res.body) {
    epDataRecv.add(res.body.length, tags);
  } else {
    epDataRecv.add(0, tags);
  }
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
 * createReports processes the test results at the end of the k6 run
 * and generates multiple report formats.
 *
 * To use this, import it in your k6 script:
 * ```
 * import createReports from './path/from/script/to/k6_utils.js';
 *
 * ...
 *
 * export function handleSummary(data) {
 *   const prefix = "some/path/to/a/file"
 *   return createReports(getPathBasename(prefix), data)
 * }
 * ```
 *
 * See: https://grafana.com/docs/k6/latest/results-output/end-of-test/custom-summary/#use-handlesummary
 *
 * @param {object} data - The k6 summary data object.
 * @returns {object} An object where keys are file paths and values are the report content.
 */
export function createReports(prefix, data) {
  console.log('Finished executing test! Generating summary reports...');

  const stdout = textSummary(data, { indent: ' ', enableColors: true });
  // Set the HTML report's title to a Space-separated Pascal Case with the given prefix
  let SummaryReportOptions = { title: prefix.toPascalCase(true) + " Run Report" };

  return {
    // 1. Write the color-enabled summary of the executed test to stdout.
    'stdout': stdout,
    // 2. Write an xml JUnit results file.
    [prefix + 'junit.xml']: jUnit(data),
    // 3. Write a JSON file with the summary of requests and metrics.
    [prefix + 'summary.json']: JSON.stringify(data, null, 2),
    // 4. Write an HTML file with the summary of request and metrics.
    [prefix + 'summary.html']: htmlReport(data, SummaryReportOptions),
    // 5. Write a custom JUnit XML file with detailed threshold results.
    [prefix + 'junit-custom.xml']: generateCustomJUnit(data),
  };
}

/**
 * A pre-configured `handleSummary` function that generates multiple reports.
 *
 * This function is a convenient wrapper around `createReports`. It automatically
 * determines the report filename prefix from the `K6_TEST` environment variable.
 *
 * To use this, import it then export it as `handleSummary`:
 * ```javascript
 * import { customHandleSummary } from './path/from/script/to/k6_utils.js';
 *
 * export const handleSummary = customHandleSummary;
 * ```
 *
 * This will generate the following files, prefixed with the test script name:
 * - `stdout`: The standard k6 text summary.
 * - `[prefix]-junit.xml`: A standard JUnit XML report.
 * - `[prefix]-summary.json`: A JSON file with the full summary data.
 * - `[prefix]-summary.html`: An HTML report.
 * - `[prefix]-junit-custom.xml`: A custom JUnit XML report focusing on thresholds.
 *
 * @param {object} data - The k6 summary data object, passed automatically by k6.
 * @returns {object} An object mapping filenames to report content, for k6 to write to disk.
 */
export function customHandleSummary(data) {
  const resultsDir = __ENV.K6_RESULTS_DIR || '';
  let prefix = __ENV.K6_TEST ? __ENV.K6_TEST.replace(/\.js$/, '') + "-" : '';
  if (resultsDir) {
    prefix = `${resultsDir}/${prefix}`;
  }
  return createReports(getPathBasename(prefix), data)
}
