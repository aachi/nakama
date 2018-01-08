export let authenticated = false
export let authUser = /** @type {AuthUser} */ (null)

const authUserItem = localStorage.getItem('auth_user')
const expiresAtItem = localStorage.getItem('expires_at')

if (authUserItem !== null && expiresAtItem !== null) {
    const expiresAt = new Date(expiresAtItem)
    const now = new Date()
    if (!isNaN(expiresAt.getDate()) && expiresAt > now) {
        try {
            authUser = JSON.parse(authUserItem)
            authenticated = true
        } catch (_) { }
    }
}

/**
 * @typedef AuthUser
 * @property {string} username
 * @property {string=} avatarUrl
 */
