#!/usr/bin/env sh

set -xe

curl -o papaparse-5.4.0.js https://raw.githubusercontent.com/mholt/PapaParse/5.4.0/papaparse.js
curl -o js-yaml-4.1.0.mjs https://raw.githubusercontent.com/nodeca/js-yaml/4.1.0/dist/js-yaml.mjs
curl -o k6-summary-0.1.0.js https://jslib.k6.io/k6-summary/0.1.0/index.js
curl -o url-1.0.0.js https://jslib.k6.io/url/1.0.0/index.js
curl -o k6-reporter-3.0.1.js https://raw.githubusercontent.com/benc-uk/k6-reporter/refs/tags/3.0.1/dist/bundle.js
