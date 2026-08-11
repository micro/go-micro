module go-micro.dev/v4~

go 1.24

// Phantom path (tilde). The natural retraction tag v4~/v1.18.2
// is an invalid git ref name, so this retraction is published as the root tag
// v1.18.2 instead — which is how this path is already keyed (its versions are
// the repo's v1.x root tags).
retract [v0.0.0, v1.18.0]
