package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	_ "modernc.org/sqlite"
	"net/http"
	"time"
)

type ExchangeRate struct {
	USD_BRL struct {
		Bid string `json:"bid"`
	} `json:"USDBRL"`
}
type CotacaoResponse struct {
	Bid string `json:"bid"`
}

type BidRepository struct {
	DB *sql.DB
}

func main() {

	http.HandleFunc("/cotacao", cotacao)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func NewBidRepository() *BidRepository {
	dbConn, err := sql.Open("sqlite", "./server/cotacao.db")
	if err != nil {
		panic(err)
	}
	_, err = dbConn.Exec("CREATE TABLE IF NOT EXISTS bids (bid TEXT)")

	if err != nil {
		panic(err)
	}
	return &BidRepository{DB: dbConn}
}

func (bidRepo *BidRepository) SaveBid(bid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := bidRepo.DB.ExecContext(ctx, "INSERT INTO bids (bid) VALUES (?)", bid)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("Timeout ao salvar cotação")
			return errors.New("request timed out")
		}
		return err
	}
	return nil
}

func getExchangeRate() (*ExchangeRate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		panic(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("Timeout ao obter cotação")
			return nil, errors.New("request timed out")
		}
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var exchangeRate ExchangeRate
	err = json.Unmarshal(body, &exchangeRate)
	if err != nil {
		return nil, err
	}
	return &exchangeRate, nil
}

func cotacao(w http.ResponseWriter, r *http.Request) {
	bidRepository := NewBidRepository()
	defer bidRepository.DB.Close()
	exchangeRate, err := getExchangeRate()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	err = bidRepository.SaveBid(exchangeRate.USD_BRL.Bid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	var cotacaoResponse = CotacaoResponse{Bid: exchangeRate.USD_BRL.Bid}
	json.NewEncoder(w).Encode(cotacaoResponse)
}
