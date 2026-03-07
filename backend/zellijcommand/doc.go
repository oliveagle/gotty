// Package zellijcommand provides a backend for gotty that uses zellij
// for persistent terminal sessions.
//
// When a client disconnects, the zellij session continues running in the
// background, allowing the user to reconnect to the same session later.
package zellijcommand
