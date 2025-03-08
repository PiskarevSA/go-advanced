package handlers

import (
	"fmt"
	"net/http"
)

const (
	docTemplate = `<!DOCTYPE html>
<title>Metrics</title>
<body>
	<table>
		<tr>
			<th>type</th>
			<th>key</th>
			<th>value</th>
		</tr>%s
	</table>
</body>
`

	rowTemplate = `
		<tr>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
		</tr>`
)

type Dumper interface {
	DumpIterator() func() (type_ string, name string, value string, exists bool)
}

// GET /
func MainPage(dumper Dumper) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		metricsIterator := dumper.DumpIterator()

		var rows string
		for {
			type_, name, value, exists := metricsIterator()
			if !exists {
				break
			}
			rows += fmt.Sprintf(rowTemplate, type_, name, value)
		}

		doc := fmt.Sprintf(docTemplate, rows)
		fmt.Println(doc)

		_, err := res.Write([]byte(doc))
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		res.Header().Set("Content-Type", "text/html")
	}
}
