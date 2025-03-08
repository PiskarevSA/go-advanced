package handlers

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
)

// GET /
func MainPage(repo Repositories) func(res http.ResponseWriter, req *http.Request) {
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

		gauge, counter := repo.Dump()

		var keys []string
		for k := range gauge {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// TODO PR #5
		// кароч странный код. Надо подумать, как упростить, чтобы не повторять
		// его. Есть еще template пакет, посмотри, как с ним сделать
		for _, k := range keys {
			// header
			lines = append(lines, "<tr>")
			// .. type
			lines = append(lines, "<td>")
			lines = append(lines, "gauge")
			lines = append(lines, "</td>")
			// key
			lines = append(lines, "<td>")
			lines = append(lines, k)
			lines = append(lines, "</td>")
			// value
			lines = append(lines, "<td>")
			lines = append(lines, fmt.Sprint(gauge[k]))
			lines = append(lines, "</td>")

			lines = append(lines, "</tr>")
		}

		keys = make([]string, 0)
		for k := range counter {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			// header
			lines = append(lines, "<tr>")
			// .. type
			lines = append(lines, "<td>")
			lines = append(lines, "counter")
			lines = append(lines, "</td>")
			// key
			lines = append(lines, "<td>")
			lines = append(lines, k)
			lines = append(lines, "</td>")
			// value
			lines = append(lines, "<td>")
			lines = append(lines, fmt.Sprint(counter[k]))
			lines = append(lines, "</td>")

			lines = append(lines, "</tr>")
		}

		lines = append(lines, "</table>")
		lines = append(lines, "</body>")
		_, err := res.Write([]byte(strings.Join(lines, "\n")))
		// TODO PR #5
		// Как будто тут надо возвращать ошибку
		if err != nil {
			log.Println(err)
		}
	}
}
