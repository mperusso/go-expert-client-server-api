package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"
	"time"
)

type Bid struct {
	Bid string `json:"bid"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		panic(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			panic("Timeout ao receber cotação do server")
		}
		panic(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		panic(fmt.Errorf("Error getting exchange rate: %s", string(body)))
	}
	if err != nil {
		panic(err)
	}
	var bid Bid
	err = json.Unmarshal(body, &bid)
	if err != nil {
		panic(err)
	}
	file, err := os.OpenFile("cotacao.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	if err != nil {
		panic(err)
	}
	template.Must(template.New("cotacao").Parse("Dólar: {{.Bid}}\n")).Execute(file, bid)
}
