package main

import (
	"fmt"
    "net/http"
    "io/ioutil"
    "strings"
)

type StockService struct {}

func main() {
	var inputOperation int
	var inputRequest string
	var budget string
	var tradeId string
	var temp float64    

	url := "http://localhost:8080/stocks"
    fmt.Println("URL:>", url)

    fmt.Println("Enter Operation to be performed (1/2)\n1. Buy Stocks\n2. Check Portfolio")
	fmt.Scanf("%d", &inputOperation)
	fmt.Scanln(&temp)

	switch inputOperation {
		case 1:			
			fmt.Printf("Please enter stock symbols and percentage with comma seperated values : ")
			input, err := fmt.Scanf("%s", &inputRequest)
			if err != nil || input != 1 {
				fmt.Println(input, err)
				return
			}
			
			fmt.Printf("Please enter total budget : ")
			fmt.Scanln(&temp)
			inputBudget, err := fmt.Scanf("%s", &budget)
			if err != nil || inputBudget != 1 {
				fmt.Println(inputBudget, err)
				return
			}	

			inputString := "{\"method\":\"StockService.PurchaseStocks\",\"params\":[{\"StockSymbolAndPercentage\":\"" + inputRequest + "\", \"Budget\":" + budget + "}],\"id\":\"1\"}"
			
		    request, err := http.NewRequest("POST", url, strings.NewReader(inputString))
		    request.Header.Set("Content-Type", "application/json")
		    client := &http.Client{}
		    response, err := client.Do(request)
		    if err != nil {
		        panic(err)
		    }
		    defer response.Body.Close()		    

		    fmt.Println("response Status:", response.Status)
		    fmt.Println("response Headers:", response.Header)
		    body, _ := ioutil.ReadAll(response.Body)
		    fmt.Println("response Body:", string(body))
		
		case 2:
			fmt.Printf("Enter Trade Id : ")
			inputTradeID, err := fmt.Scanf("%s", &tradeId)
			if err != nil || inputTradeID != 1 {
				fmt.Println(inputTradeID, err)
				return
			}

			inputString := "{\"method\":\"StockService.ShowPortfolio\",\"params\":[{\"TradeId\":" + tradeId + "}],\"id\":\"1\"}"
			fmt.Println("Input : ", inputString)
			request, err := http.NewRequest("POST", url, strings.NewReader(inputString))
			request.Header.Set("Content-Type", "application/json")
		    client := &http.Client{}
		    response, err := client.Do(request)
		    if err != nil {
		        panic(err)
		    }
		    defer response.Body.Close()		    

		    fmt.Println("response Status:", response.Status)
		    fmt.Println("response Headers:", response.Header)
		    body, _ := ioutil.ReadAll(response.Body)
		    fmt.Println("response Body:", string(body))

		default:
			fmt.Println("Invalid Operation!")
	}	
}