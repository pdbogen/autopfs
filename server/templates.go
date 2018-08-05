package main

import "html/template"

var TemplateRoot = template.New("")
var HeaderTemplate = template.Must(TemplateRoot.New("header").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8"/>
<title>AutoPFS{{if .Title}}: {{.Title}}{{end}}</title>
<style type='text/css'>
tr:nth-of-type(even) {
	background: #c0c0c0;
}
</style>
</head>
<body>`))
var FooterTemplate = template.Must(TemplateRoot.New("footer").Parse(`
</body>
</html>
`))
var HtmlTemplate = template.Must(TemplateRoot.New("html").Parse(`
{{template "header"}}
<a href="/csv?id={{.id}}">Download as CSV</a> or <a href="/status?id={{.id}}&view=true">View the Job Log</a>
<table>
	<thead>
		<tr>
			{{range .Headers}}<th>{{.}}</th>{{end}}
		</tr>
	</thead>
	<tbody>
		{{range .Rows}}
		<tr>
			{{range .}}<td>{{.}}</td>{{end}}
		</tr>
		{{end}}
	</tbody>
</table>
{{template "footer"}}
`))

var StatusTemplate = template.Must(TemplateRoot.New("status").Parse(`
{{template "header"}}
{{if .Job.Done}}
This job is complete! <a href="/html?id={{.Job.JobId}}">View the Results</a>.<br/>
{{else}}
This can potentially take a few minutes. Please hold tight.<br/>
{{end}}
Status: {{.Job.State}}<br/>
Job Log:<br/>
<ul>
{{range .Job.Messages}}
<li><pre style='white-space: pre-wrap'>{{.}}</pre></li>
{{end}}
</ul>
{{template "footer"}}
`))
