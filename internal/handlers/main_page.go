package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

type Dumper interface {
	DumpIterator() func() (type_ string, name string, value string, exists bool)
}

// GET /
func MainPage(dumper Dumper) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "text/html")
		// TODO PR #5
		// Если ты хочешь использовать какой-то html шаблон, посмотри в сторону
		// go.embed или же просто создай шаблон и засунь туда значения через
		// одинарные кавычки
		//
		// `
		// <title>Title</title> `

		var lines []string
		lines = append(lines, "<!DOCTYPE html>")
		lines = append(lines, "<title>Metrics</title>")
		lines = append(lines, "<body>")
		lines = append(lines, "<table>")

		// header
		lines = append(lines, "<tr>")
		// .. type
		lines = append(lines, "<th>")
		lines = append(lines, "type")
		lines = append(lines, "</th>")
		// .. key
		lines = append(lines, "<th>")
		lines = append(lines, "key")
		lines = append(lines, "</th>")
		// .. value
		lines = append(lines, "<th>")
		lines = append(lines, "value")
		lines = append(lines, "</th>")

		lines = append(lines, "</tr>")

		// TODO PR #5
		// кароч странный код. Надо подумать, как упростить, чтобы не повторять
		// его. Есть еще template пакет, посмотри, как с ним сделать
		metricsIterator := dumper.DumpIterator()

		for {
			type_, name, value, exists := metricsIterator()
			if !exists {
				break
			}
			// metric
			lines = append(lines, "<tr>")
			// .. type
			lines = append(lines, "<td>")
			lines = append(lines, type_)
			lines = append(lines, "</td>")
			// .. key
			lines = append(lines, "<td>")
			lines = append(lines, name)
			lines = append(lines, "</td>")
			// .. value
			lines = append(lines, "<td>")
			lines = append(lines, fmt.Sprint(value))
			lines = append(lines, "</td>")

			lines = append(lines, "</tr>")
		}

		lines = append(lines, "</table>")
		lines = append(lines, "</body>")
		_, err := res.Write([]byte(strings.Join(lines, "\n")))
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
	}
}
