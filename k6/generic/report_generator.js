import { customHandleSummary } from './k6_utils.js';

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
