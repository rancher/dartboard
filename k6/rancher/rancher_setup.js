import {getCookies, firstLogin, logout} from "./rancher_utils.js";

export const options = {
    insecureSkipTLSVerify: true,
}

export default function main() {
    const baseUrl = __ENV.BASE_URL
    const bootstrapPassword = __ENV.BOOTSTRAP_PASSWORD
    const password = __ENV.PASSWORD

    const cookies = getCookies(baseUrl)

    firstLogin(baseUrl, cookies, bootstrapPassword, password)

    logout(baseUrl, cookies)
}
