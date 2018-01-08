const template = document.createElement('template')
template.innerHTML = `
<div class="container">
    <h1>Notifications</h1>
    <div class="articles">
        <article class="notification">
            <span>
                <a href="/users/jane_doe">jane_doe</a> liked <a href="/posts/1">your post</a>
            </span>
            <time>2m</time>
        </article>
        <article class="notification">
            <span>
                <a href="/users/jane_doe">jane_doe</a> liked <a href="/posts/1">your post</a>
            </span>
            <time>4m</time>
        </article>
        <article class="notification read">
            <span>
                <a href="/users/jane_doe">jane_doe</a> liked <a href="/posts/1">your post</a>
            </span>
            <time>5m</time>
        </article>
    </div>
</div>
`

export default function () {
    const page = /** @type {DocumentFragment} */ (template.content.cloneNode(true))
    return page
}
