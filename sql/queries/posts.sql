-- name: CreatePost :exec
INSERT INTO posts (title, url, description, published_at, feed_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
ON CONFLICT (url) DO NOTHING;

-- name: GetPostsUser :many
SELECT posts.*
FROM posts
JOIN feed_follows ON feed_follows.feed_id = posts.feed_id
WHERE feed_follows.user_id = $1
ORDER BY published_at DESC NULLS LAST
LIMIT $2;
