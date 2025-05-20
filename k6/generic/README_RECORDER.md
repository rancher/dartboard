# How to create tests using k6 recorder

 - install Chrome ([Firefox has a bug with self-signed certificates on Web Sockets](https://bugzilla.mozilla.org/show_bug.cgi?id=594502) as of March 2023)
 - follow instructions in [k6 docs about Browser Recorder](https://k6.io/docs/test-authoring/recording-a-session/browser-recorder/) to install the recorder
 - start recording
 - navigate to Rancher's address
 - perform the operations under test
 - stop the recording. Select "script editor" and copy the Javascript code from there
 - use the following find/replace regexps to replace hardcoded URLs:
```
Find:
'(.*?)https://(?:.+?)/(.*?)'

Replace:
`$1\${baseUrl}/$2`

Find:
`(.*?)https://(?:.+?)/(.*?)`

Replace:
`$1\${baseUrl}/$2`

Find:
`(.*)user\-[a-z0-9]+(.*)`

Replace:
`$1\${userId}$2`
```
- use the following find/replace regexps to remove unimportant headers:
```
Find:
^ +'((upgrade-insecure-requests)|(sec-.+)|(x-api-csrf))':.+,\n

Replace:

Find:
 +headers\: \{\n +\}\,\n

Replace:

Find:
, +\{\n +\}

Replace:

```
- add the following parameter to the option map to all get/put/post/delete methods requiring authentication
```
cookies: cookies,
```
- use the following find/replace regexp to replace timestamps:
```
Find:
"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z"

Replace:
timestamp
```
- replace string JSON payloads with objects eg. `'{"my":"payload"}'` with `JSON.stringify({"my":"payload"})`
- remove `group` calls
- replace all code before the default exported function with:
```javascript
import { check, fail } from 'k6';
import http from 'k6/http'

const baseUrl = __ENV.BASE_URL
const bootstrapPassword = __ENV.BOOSTRAP_PASSWORD
const password = __ENV.PASSWORD
```
