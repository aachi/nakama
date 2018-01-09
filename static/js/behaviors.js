import http from './http.js'
import { likesMsg, followMsg, followersMsg } from './utils.js'

/**
 * Connects a like button to the server API.
 *
 * @param {HTMLButtonElement} button
 * @param {string} resource
 */
export function likeable(button, resource) {
    button.addEventListener('click', () => {
        button.disabled = true
        http.post(`/api/${resource}/toggle_like`).then(payload => {
            button.textContent = String(payload.likesCount)
            button.classList[payload.liked ? 'add' : 'remove']('liked')
            button.setAttribute('aria-label', likesMsg(payload.likesCount))
            button.setAttribute('aria-checked', String(payload.liked))
        }).catch(console.error).then(() => {
            button.disabled = false
        })
    })
}

/**
 * Connects a follow button to the server API.
 *
 * @param {HTMLButtonElement} button
 * @param {string} username
 */
export function followable(button, username) {
    const followersEl = button.parentElement.parentElement.querySelector('.followers-count')
    button.addEventListener('click', () => {
        button.disabled = true
        http.post(`/api/users/${username}/toggle_follow`).then(payload => {
            button.textContent = followMsg(payload.followingOfMine)
            if (followersEl !== null) {
                followersEl.textContent = followersMsg(payload.followersCount)
            }
        }).catch(console.error).then(() => {
            button.disabled = false
        })
    })
}

export function spoileable(button) {
    const togglerWrapper = button.parentElement
    const articleContent = /** @type {HTMLElement} */ (button.parentElement.parentElement.querySelector('.content'))
    button.addEventListener('click', () => {
        togglerWrapper.remove()
        articleContent.hidden = false
    })
}
