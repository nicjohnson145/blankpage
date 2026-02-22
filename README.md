# blankpage

A barebones OPDS server focusing on simplicity &amp; less clicks

## Why?

I started off with [Calibre-Web](https://github.com/janeczku/calibre-web) and all things considered, its pretty great.
However I wanted something a bit simpler, and with the ability to upload books without transferring files around and clicking
in web pages. I wanted it to have a real API. Plus building things is cool, and learning about the OPDS protocol is
neat.

## "Man, this UI sucks!"

Yeah. I know. I'm a backend guy. I suck at frontend work really bad.

## TODO:

* [ ] Thumbnails
* [ ] AuthN/AuthZ UI pages
* [ ] UI upload support
* [ ] More "filters", both in UI & OPDS

## Running

Use your favorite orchestrator, images are available on Github's image registry, and pre-compiled binaries are available
in releases. Looking at the `server` command in `Taskfile.dist.yml` should give a decent overview. Basically:

* Configure via env-vars, see [internal/svcconfig/config.go](https://github.com/nicjohnson145/blankpage/blob/main/internal/svcconfig/config.go)
  for details of what the various settings do
* Its a static binary with the UI bundled in. Just run it. Point it to a Postgres instance if you want your changes to
  stick around.
