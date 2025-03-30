package handlers

import (
	"fmt"
	"net/http"
)

const (
	MainPagePattern = `/`
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

type DumperUsecase interface {
	DumpIterator() func() (type_ string, name string, value string, exists bool)
}

// MainPageHandler handles endpoint: GET /
// request: none
// response	type: "text/html", body: html document containing dumped metrics
type MainPageHandler struct {
	Dumper DumperUsecase
}

func (h *MainPageHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	metricsIterator := h.Dumper.DumpIterator()

	var rows string
	for {
		type_, name, value, exists := metricsIterator()
		if !exists {
			break
		}
		rows += fmt.Sprintf(rowTemplate, type_, name, value)
	}

	doc := fmt.Sprintf(docTemplate, rows)

	res.Header().Set("Content-Type", "text/html")
	_, err := res.Write([]byte(doc))
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
