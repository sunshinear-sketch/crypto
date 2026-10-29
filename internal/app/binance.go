package app

import (
 "context"
 "encoding/json"
 "fmt"
 "net/http"
 "net/url"
 "strconv"
 "strings"
 "time"
)
type Market struct {
 Symbol string `json:"symbol"`
 Price float64 `json:"price"`
 Volume24h float64 `json:"volume24h"`
 RSI4h float64 `json:"rsi4h"`
 RSI1d float64 `json:"rsi1d"`
 Low90d float64 `json:"low90d"`
 Distance90d float64 `json:"distance90d"`
 FundingRate float64 `json:"fundingRate"`
 Score float64 `json:"score"`
 UpdatedAt time.Time `json:"updatedAt"`
}
type Binance struct { Base string; Client *http.Client }
func (b Binance) get(ctx context.Context,path string,out any) error {
 req,err:=http.NewRequestWithContext(ctx,"GET",strings.TrimSuffix(b.Base,"/")+path,nil)
 if err!=nil{return err}
 resp,err:=b.Client.Do(req);if err!=nil{return err}
 defer resp.Body.Close()
 if resp.StatusCode!=200{return fmt.Errorf("binance %s: HTTP %d",path,resp.StatusCode)}
 return json.NewDecoder(http.MaxBytesReader(nil,resp.Body,8<<20)).Decode(out)
}
func (b Binance) Symbols(ctx context.Context)([]string,error){
 var v struct{Symbols []struct{Symbol string `json:"symbol"`; Status string `json:"status"`; ContractType string `json:"contractType"`; QuoteAsset string `json:"quoteAsset"`} `json:"symbols"`}
 if err:=b.get(ctx,"/fapi/v1/exchangeInfo",&v);err!=nil{return nil,err}
 a:=[]string{}
 for _,s:=range v.Symbols{if s.Status=="TRADING"&&s.ContractType=="PERPETUAL"&&s.QuoteAsset=="USDT"{a=append(a,s.Symbol)}}
 return a,nil
}
func (b Binance) Tickers(ctx context.Context)(map[string]struct{Price,Volume float64},error){
 var v []struct{Symbol string `json:"symbol"`;LastPrice string `json:"lastPrice"`;QuoteVolume string `json:"quoteVolume"`}
 if err:=b.get(ctx,"/fapi/v1/ticker/24hr",&v);err!=nil{return nil,err}
 m:=map[string]struct{Price,Volume float64}{}
 for _,t:=range v{p,_:=strconv.ParseFloat(t.LastPrice,64);q,_:=strconv.ParseFloat(t.QuoteVolume,64);m[t.Symbol]=struct{Price,Volume float64}{p,q}}
 return m,nil
}
func (b Binance) Funding(ctx context.Context)(map[string]float64,error){
 var v []struct{Symbol string `json:"symbol"`;LastFundingRate string `json:"lastFundingRate"`}
 if err:=b.get(ctx,"/fapi/v1/premiumIndex",&v);err!=nil{return nil,err}
 m:=map[string]float64{}
 for _,t:=range v{f,_:=strconv.ParseFloat(t.LastFundingRate,64);m[t.Symbol]=f}
 return m,nil
}
func (b Binance) Candles(ctx context.Context,symbol,interval string,limit int)([]float64,[]float64,error){
 q:=url.Values{};q.Set("symbol",symbol);q.Set("interval",interval);q.Set("limit",strconv.Itoa(limit))
 var v [][]json.RawMessage
 if err:=b.get(ctx,"/fapi/v1/klines?"+q.Encode(),&v);err!=nil{return nil,nil,err}
 closes,lows:=[]float64{},[]float64{}
 for _,row:=range v{
  if len(row)<5{continue}
  var c,l string
  if json.Unmarshal(row[4],&c)!=nil||json.Unmarshal(row[3],&l)!=nil{continue}
  cv,e1:=strconv.ParseFloat(c,64);lv,e2:=strconv.ParseFloat(l,64)
  if e1==nil&&e2==nil{closes=append(closes,cv);lows=append(lows,lv)}
 }
 if len(closes)<20{return nil,nil,fmt.Errorf("not enough candles for %s",symbol)}
 return closes,lows,nil
}
