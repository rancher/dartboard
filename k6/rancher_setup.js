import {getCookies, firstLogin, createImportedCluster, logout} from "./rancher_utils.js";


export const options = {
    insecureSkipTLSVerify: true,
}

export default function main() {
    const baseUrl = __ENV.BASE_URL
    const bootstrapPassword = __ENV.BOOTSTRAP_PASSWORD
    const password = __ENV.PASSWORD
    const importedClusterNames = __ENV.IMPORTED_CLUSTER_NAMES.split(",")

    const cookies = getCookies(baseUrl)

    firstLogin(baseUrl, cookies, bootstrapPassword, password)

    for (const name in importedClusterNames) {
        createImportedCluster(baseUrl, cookies, name)
    }

    logout(baseUrl, cookies)
}
