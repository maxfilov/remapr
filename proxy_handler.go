package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

type ProxyHandler struct {
	method    string
	path      string
	transform JsonTransform
}

type Req struct {
	Ids []int `json:"entityIds"`
}

func (p ProxyHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var req Req
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(err.Error()))
		return
	}
	var newreq *http.Request = request.Clone(request.Context())
	var client http.Client = http.Client{}
	newreq.Method = p.method
	newreq.URL.Path = p.path
	resp, err := client.Do(newreq)
	if err != nil {
		writer.WriteHeader(http.StatusBadGateway)
		_, _ = writer.Write([]byte(err.Error()))
		return
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		writer.WriteHeader(http.StatusBadGateway)
		_, _ = writer.Write([]byte(err.Error()))
		return
	}
	newResponse, err := p.transform(respBytes, convertIds(req.Ids))
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(err.Error()))
		return
	}
	_, err = writer.Write(newResponse)
	if err != nil {
		return
	}
}

func convertIds(ids []int) map[string]int {
	var result map[string]int = make(map[string]int)
	for _, id := range ids {
		result[strconv.Itoa(id)] = 0
	}
	return result
}
