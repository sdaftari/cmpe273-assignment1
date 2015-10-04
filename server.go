package main

import (
	"fmt"
    "net/http"
    "github.com/gorilla/rpc"
    "io/ioutil"
    "github.com/gorilla/rpc/json"
    "io"
    "strings"
    "os"
    "strconv"
    "errors"
)

type StockService struct {}

//Static part of Yahoo API for each customer
const (
    QuotesUrl = "http://download.finance.yahoo.com/d/quotes.csv?s="
)

var TradeIdForCustomer int
var budget float64
var isNewPurchase bool

var objStock *StockParameters
var objAllStocks StockStructre
var listStocks []StockParameters

var objStockDistribution *StockDistributionParameters
var objAllStockDistribution StockDistribution
var listDistributedStocks []StockDistributionParameters

var listResponse []StockDistribution
var objAllResponse AllResponses

var stockResponse StockResponse

//Structure of Request containing input parameters from user
//Ex: Input : GOOG:50%,YHOO:10% and Budget : 500
type StockRequest struct {
    StockSymbolAndPercentage string
    Budget float64
}

type StockResponse struct {
    TradeId int
    Stocks string
    UnvestedAmount float64
}

//Structure to store a stock status having symbol, 
//percentage of budget and Real time stock amount from yahoo API
type StockParameters struct {
    Symbol string
    Percent float64
    StockAmount float64
}

//Strcure containing array of above stock parameters
type StockStructre struct {
    StockParams []StockParameters
}

//Structure used to save number and amount of stocks for each symbol
type StockDistributionParameters struct {
    Symbol string
    NumberOfStocksForSymbol int
    AmountOfStockSymbol float64
}

//Structure containing TradeId, Unvested amount and stocks information 
//for each customer per trade transaction
type StockDistribution struct {
    TradeId int
    UninvestedAmount float64
    StockDistributionArray []StockDistributionParameters
}

//In memory structure used to save all stocks
type AllResponses struct {
    CustomerResponses []StockDistribution
}

type RequestPortfolio struct {
    TradeId int
}

type ResponsePortfolio struct {
    CurrentStocks string
    CurrentMarketValue float64
    CurrentUnvestedAmount float64
}

func (t *StockService) ShowPortfolio(r *http.Request, args *RequestPortfolio, reply *ResponsePortfolio) error {
    var objStockSymbols string
    isNewPurchase = false
    
    previousStockStatus, err := getStockDistributionForTradeId(args.TradeId)
    checkError(err)

    for _, i := range previousStockStatus.StockDistributionArray {
        objStockSymbols = objStockSymbols + "," + i.Symbol
    }

    startsWith := strings.HasPrefix(objStockSymbols, ",")
    if startsWith {
        objStockSymbols = strings.TrimPrefix(objStockSymbols, ",")
    }  

    //Pass the input values of stock symbols and 
    //get the real time stocks in csv
    response, err := getCsv(objStockSymbols)
    checkError(err)

    //Read the contents of response
    contents, err := ioutil.ReadAll(response)   
    response.Close()
    checkError(err)

    data := strings.Split(string(contents), "\n")

    objStock = new(StockParameters)
    listStocks = []StockParameters{}
    objAllStocks = StockStructre{listStocks}

    for i := 0; i < len(data); i++ {
        if data[i] != "" {
            symbolsAndPercentage := strings.Split(data[i], ",")        
            (*objStock).Symbol = symbolsAndPercentage[0][1:len(symbolsAndPercentage[0]) - 1]
            (*objStock).Percent = 0.0

            objAllStocks.AddItem(*objStock)   
        }
    }

    createStockStructure(string(contents)) 

    currentStockStatus := getStockStatus()

    objTest := getCurrentMarketStatus(previousStockStatus, currentStockStatus)

    *reply = *objTest

    return nil  
}

func getCurrentMarketStatus(previousStockStatus *StockDistribution, currentStockStatus StockDistribution) *ResponsePortfolio {
    var stockStatus string
    var amountSign string
    var amountDifference float64
    var currentTotalAmount float64

    objResponsePortfolio := new(ResponsePortfolio)

    for i := 0; i < len(previousStockStatus.StockDistributionArray); i++ {
        for j := 0; j < len(currentStockStatus.StockDistributionArray); j++ {
            if previousStockStatus.StockDistributionArray[i].Symbol == currentStockStatus.StockDistributionArray[j].Symbol {
                symbol := currentStockStatus.StockDistributionArray[j].Symbol
                currentValue := currentStockStatus.StockDistributionArray[j].AmountOfStockSymbol
                currentTotalAmount = currentTotalAmount + currentValue * float64(previousStockStatus.StockDistributionArray[i].NumberOfStocksForSymbol)
                amountDifference := previousStockStatus.StockDistributionArray[i].AmountOfStockSymbol - currentStockStatus.StockDistributionArray[j].AmountOfStockSymbol

                if amountDifference == 0.0 {
                    amountSign = ""
                } else if amountDifference > 0.0 {
                    amountSign = "+"
                } else {
                    amountSign = "-"
                }
                stockStatus = stockStatus + symbol + ":" + strconv.Itoa(previousStockStatus.StockDistributionArray[i].NumberOfStocksForSymbol) + ":" + amountSign + "$" + strconv.FormatFloat(currentValue, 'f', -1, 64) + ","
            }
        }
    }

    endsWith := strings.HasSuffix(stockStatus, ",")
    if endsWith {
        stockStatus = strings.TrimSuffix(stockStatus, ",")
    }

    (objResponsePortfolio).CurrentMarketValue = currentTotalAmount
    (objResponsePortfolio).CurrentUnvestedAmount = previousStockStatus.UninvestedAmount + amountDifference
    (objResponsePortfolio).CurrentStocks = stockStatus

    return objResponsePortfolio
}

func getStockDistributionForTradeId(tradeId int) (*StockDistribution, error) {
    for i := 0; i < len(objAllResponse.CustomerResponses); i++ {
        if objAllResponse.CustomerResponses[i].TradeId == tradeId {
            objStock := &objAllResponse.CustomerResponses[i]
            return objStock, nil
        }
    }

    return nil, errors.New("Stock Information is not found!")
}

func (t *StockService) PurchaseStocks(r *http.Request, args *StockRequest, reply *StockResponse) error {
    budget = args.Budget
    isNewPurchase = true
    var stocks string

    //Get the stock symbols from input in comma seperated form, Ex: GOOG,YHOO
    listInputParams := getListOfInputParameters(args.StockSymbolAndPercentage)

    if strings.EqualFold(listInputParams, "Invalid Input") {
        err := errors.New("Invalid Distribution of Budget. Please enter valid percentage!")
        checkError(err)
    } else {
        //Pass the input values of stock symbols and 
        //get the real time stocks in csv
        response, err := getCsv(listInputParams)
        checkError(err)

        //Read the contents of response
        contents, err := ioutil.ReadAll(response)
        response.Close()
        checkError(err) 

        createStockStructure(string(contents))

        stockDistribution := getStockStatus()        

        for _, i := range stockDistribution.StockDistributionArray {
            stocks = stocks + i.Symbol + ":" + strconv.Itoa(i.NumberOfStocksForSymbol) + ":$" + strconv.FormatFloat(i.AmountOfStockSymbol, 'f', -1, 64) + ","
        }

        stockResponse = StockResponse{stockDistribution.TradeId, stocks, stockDistribution.UninvestedAmount}

        fmt.Println(stockResponse)
        
        *reply = stockResponse
    }

    return nil
}

/*
Gets the list of symbols from user input in comma seperated string value
Ex: GOOG,YHOO
Also craetes a structure to save symbol and stock percantage value
Ex: Symbol : GOOG
    Percent : 50
Returns comma seperated input symbols
*/
func getListOfInputParameters(stockSymbol string) string {
    objStock = new(StockParameters)
    listStocks = []StockParameters{}
    objAllStocks = StockStructre{listStocks}

    var inputSymbols string
    var totalPercentage float64
    
    data := strings.Split(stockSymbol, ",") 
    
    //Create a structure object (objStock) with Symbol and Percent
    //Add each object to an array objAllStocks to contain symbol, percent for each customer
    for i := 0; i < len(data); i++ {
        symbolsAndPercentage := strings.Split(data[i], ":")
        (*objStock).Symbol = symbolsAndPercentage[0]

        endsWith := strings.HasSuffix(symbolsAndPercentage[1], "%")
        if endsWith {
            symbolsAndPercentage[1] = strings.TrimSuffix(symbolsAndPercentage[1], "%")
        }
        percent, err := strconv.ParseFloat(symbolsAndPercentage[1], 64)
        checkError(err)
        (*objStock).Percent = percent

        objAllStocks.AddItem(*objStock)     
    }

    //Find the distribution of percentage
    for _, i := range objAllStocks.StockParams {
        totalPercentage = totalPercentage + i.Percent
    }

    //Throw error if the total percentage distribution > 100
    if(totalPercentage > 100.0) {
        err := errors.New("Invalid Distribution of Budget. Please enter valid percentage!")
        checkError(err)
    } else {
        for _, i := range objAllStocks.StockParams {
            inputSymbols = inputSymbols + "," + i.Symbol
        }

        startsWith := strings.HasPrefix(inputSymbols, ",")
        if startsWith {
            inputSymbols = strings.TrimPrefix(inputSymbols, ",")
        }           
    }

    return inputSymbols
}

/*Takes comma seperated string of input symbols and get the csv from yahoo API
Returns response body from Yahoo API containing symbol name and real time value
*/
func getCsv(inputSymbols string) (io.ReadCloser, error) {
    url := QuotesUrl + inputSymbols + "&f=sa"
    
    fmt.Println("csv: firing HTTP GET at ", url)
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }

    return resp.Body, nil
}

/*
Takes symbol and real time value of stock for each symbol seperated by newline
and update stock structure to contain real time stock amount along with symbol and percentage
*/
func createStockStructure(contents string) {
    listOfSymbolsAndStockValue := strings.Split(contents, "\n")
    
    for i := 0; i < len(listOfSymbolsAndStockValue); i++ {
        if listOfSymbolsAndStockValue[i] != "" {
            symbolAndStockAmount := strings.Split(listOfSymbolsAndStockValue[i], ",")
            symbol := symbolAndStockAmount[0][1:len(symbolAndStockAmount[0]) - 1]
            for j := 0; j < len(objAllStocks.StockParams); j++ {
                if strings.EqualFold(objAllStocks.StockParams[j].Symbol, symbol) {
                    amount, err := strconv.ParseFloat(symbolAndStockAmount[1], 64)
                    checkError(err)
                    objAllStocks.StockParams[j].StockAmount = amount
                }
            }
        }
    }  
}

/*
Calculates budgetwise number of stocks, unvested amount and saves in array
Also creates in-memory structure to save information of all customers
*/
func getStockStatus() StockDistribution {
    validTrade := false
    unvestedAmout := budget

    objStockDistribution = new(StockDistributionParameters)
    listDistributedStocks = []StockDistributionParameters{}
    objAllStockDistribution = StockDistribution{TradeIdForCustomer, unvestedAmout, listDistributedStocks}

    //Calculate amount and number of stocks foe each symbol based on percentage and total budget
    for i := 0; i < len(objAllStocks.StockParams); i++ {
        (*objStockDistribution).Symbol = objAllStocks.StockParams[i].Symbol

        totalBudgetForSymbol := (objAllStocks.StockParams[i].Percent * budget)/100

        (*objStockDistribution).NumberOfStocksForSymbol = int(totalBudgetForSymbol/objAllStocks.StockParams[i].StockAmount)

        (*objStockDistribution).AmountOfStockSymbol = objAllStocks.StockParams[i].StockAmount
        unvestedAmout = unvestedAmout - float64((*objStockDistribution).NumberOfStocksForSymbol) * (*objStockDistribution).AmountOfStockSymbol 

        objAllStockDistribution.AddStockDistributionItem(*objStockDistribution)
    }

    //Save total unvested amount after purchasing all stocks
    objAllStockDistribution.UninvestedAmount = unvestedAmout

    if isNewPurchase {
        if unvestedAmout >= 0.0 {
            TradeIdForCustomer = TradeIdForCustomer + 1     
            validTrade = true
        } else {
            err := errors.New("Please invest more budget to purchase the stocks!")
            checkError(err)
        }

        objAllStockDistribution.TradeId = TradeIdForCustomer

        if validTrade {
            objAllResponse.AddResponse(objAllStockDistribution)
        }
    } else {
        objAllStockDistribution.TradeId = 0
    }

    return objAllStockDistribution
}

/*Store trade information (Trade Id, all stocks, Unvested amount) for each customer for in-memory cache
Returns array of above information for all customers
*/
func (stockResponse *AllResponses) AddResponse(customerTradeResponse StockDistribution) []StockDistribution {
    stockResponse.CustomerResponses = append(stockResponse.CustomerResponses, customerTradeResponse)
    return stockResponse.CustomerResponses
}

/*
Store Tradewise information (symbols, amount, number of stocks purchased) for each customer per transaction
Returns array of all stock information
*/
func (stockDistribution *StockDistribution) AddStockDistributionItem(indivisualStock StockDistributionParameters) []StockDistributionParameters {
    stockDistribution.StockDistributionArray = append(stockDistribution.StockDistributionArray, indivisualStock)
    return stockDistribution.StockDistributionArray
}

/*
Takes object of StockParameter structure and add in array to list all symbol information for each customer
Returns array of stock information
*/
func (stock *StockStructre) AddItem(item StockParameters) []StockParameters {
    stock.StockParams = append(stock.StockParams, item)
    return stock.StockParams
}

/*
Checks for the error and exit if any
*/
func checkError(err error) {
    if err != nil {
        fmt.Println("Fatal error ", err.Error())
        os.Exit(1)
    }
}

func main() {
    //Initialize in-memory array of structure
    listResponse = []StockDistribution{}
    objAllResponse = AllResponses{listResponse}

    fmt.Println("Starting Server")
	s := rpc.NewServer()
    s.RegisterCodec(json.NewCodec(), "application/json")
    s.RegisterService(new(StockService), "")
    http.Handle("/stocks", s)
    http.ListenAndServe(":8080", nil)
}

