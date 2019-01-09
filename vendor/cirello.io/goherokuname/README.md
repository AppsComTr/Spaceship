# goherokuname

[![Build Status](https://travis-ci.org/ucirello/goherokuname.svg?branch=master)](https://travis-ci.org/ucirello/goherokuname)

Heroku-like Random Names in Go

To install use:

`go get -f -u cirello.io/goherokuname/...`

It contais three dictionaries, one small, one big and one of simple words. If you want to compile with the big one, you are going to need to use build tags:

`go build -tags herokuComplete`

or

`go get -f -u -tags 'herokuComplete' cirello.io/goherokuname/...`

If you want the one with simple words:

`go build -tags herokuSimple`

or

`go get -f -u -tags 'herokuSimple' cirello.io/goherokuname/...`


The big dictionary is actually from Wordnet of Princeton. Please, be sure to agree with their license before using it.

See documentation at [http://godoc.org/cirello.io/goherokuname](http://godoc.org/cirello.io/goherokuname).
