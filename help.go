package main

const (
	exampleTemplate = `
EXAMPLES:
   # Share your terminal as a web application (default port: 13562)
   gotty bash

   # Use custom port
   gotty --port 8080 bash

   # Enable basic authentication
   gotty --credential user:pass bash

   # Use custom title format
   gotty --title-format "gotty: {{.command}}" bash
`
)

var helpTemplate = `NAME:
   {{.Name}} - {{.Usage}}

USAGE:
   {{.Name}} [options] <command> [<arguments...>]

VERSION:
   {{.Version}}

OPTIONS:
{{range .VisibleFlags}}
   {{range $i, $name := .Names}}{{if $i}}, {{end}}{{$name}}{{end}}  {{.Usage}}
{{end}}
` + exampleTemplate
