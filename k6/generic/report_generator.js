import { customHandleSummary } from './k6_utils.js';

/*
This module is responsible for generating custom reports from the k6 summary data. 
It reads the summary JSON file specified by the K6_SUMMARY_JSON_FILE environment variable, 
processes the data, and generates custom reports based on the test results. The handleSummary 
function is called by k6 at the end of the test execution to generate the reports.

Can be used ad-hoc to generate custom reports based on the summary data.
*/

const summaryPath = __ENV.K6_SUMMARY_JSON_FILE;

// Load summary data during initialization
let data = null;
if (summaryPath) {
    try {
        data = JSON.parse(open(summaryPath));
    } catch (e) {
        console.error(`Failed to parse summary JSON from ${summaryPath}: ${e.message}`);
    }
}

export function handleSummary() {
    if (!data) {
        console.log("No summary data available to generate reports.");
        return {};
    }
    return customHandleSummary(data);
}

export default function() {}
