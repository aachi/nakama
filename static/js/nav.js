import { authenticated, authUser } from './auth.js'
import http from './http.js'

const nav = document.getElementById('nav')
nav.className = 'app-nav'
nav.innerHTML = `
    <a href="/">Home</a>
    ${authenticated ? `
        <a href="/notifications" id="notifications-link">Notifications</a>
    ` : ''}
    <a href="/search">Search</a>
    ${authenticated ? `
        <a href="/users/${authUser.username}">Profile</a>
    ` : ''}
`

if (authenticated && location.pathname !== '/notifications') {
    http.get('/api/check_unread_notifications').then(unread => {
        if (unread) {
            nav.querySelector('#notifications-link').classList.add('unread')
        }
    }).catch(console.error)
}
