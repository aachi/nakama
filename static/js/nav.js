import { authenticated, authUser } from './auth.js'

const nav = document.getElementById('nav')
nav.className = 'app-nav'
nav.innerHTML = `
    <a href="/">Home</a>
    <a href="/search">Search</a>
    ${authenticated ? `
        <a href="/users/${authUser.username}">Profile</a>
    ` : ''}
`
