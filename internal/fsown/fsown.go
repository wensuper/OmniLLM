// Package fsown says who owns a file, so a program trusts only files and
// sockets its own user (or, for system programs, root) made: never one
// another local user placed where it looks, say in /tmp.
package fsown
