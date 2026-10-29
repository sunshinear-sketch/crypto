package app

import (
 "context"
 "crypto/subtle"
 "embed"
 "encoding/json"
 "errors"
 "fmt"
 "log"
 "math"
 "net/http"
 "net/smtp"
 "os"
 "sort"
 "strconv"
 "strings"
 "sync"
 "time"
)

//go:embed web/*
var assets embed.FS

type Config struct{ Addr, DatabaseURL, BinanceBase, ScanInterval, SMTPHost, SMTPPort, SMTPUser, SMTPPass, SMTPFrom, APIKey string }
type Subscription struct {
 ID int64 `json:"id"`
 Email string `json:"email"`
 MaxRSI float64 `json:"maxRsi"`
 MaxDistance float64 `json:"maxDistance"`
 MinVolume float64 `json:"minVolume"`
 Active bool `json:"active"`
}
type App struct {
 cfg Config
 bin Binance
 mu sync.RWMutex
 markets []Market
 scanned time.Time
 scanning bool
 subscribers []Subscription
}
type snapshot struct {
 Markets []Market `json:"markets"`
 Scanned time.Time `json:"scanned"`
 Subscribers []Subscription `json:"subscribers"`
}
func(a *App) save(){
 a.mu.RLock();v:=snapshot{a.markets,a.scanned,a.subscribers};a.mu.RUnlock()
 b,e:=json.Marshal(v);if e!=nil{return}
 if os.WriteFile("screener-data.json.tmp",b,0600)==nil{_ = os.Rename("screener-data.json.tmp","screener-data.json")}
}
func(a *App) load(){
 b,e:=os.ReadFile("screener-data.json");if e!=nil{return}
 var v snapshot
 if json.Unmarshal(b,&v)==nil{a.markets=v.Markets;a.scanned=v.Scanned;a.subscribers=v.Subscribers}
}
func(a *App) authorized(r *http.Request)bool{
 v:=strings.TrimPrefix(r.Header.Get("Authorization"),"Bearer ")
 return a.cfg.APIKey!=""&&subtle.ConstantTimeCompare([]byte(v),[]byte(a.cfg.APIKey))==1
}
func jsonReply(w http.ResponseWriter,status int,v any){
 w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)
}
func Run(cfg Config)error{
 if cfg.APIKey==""{log.Print("WARNING: ADMIN_API_KEY not configured; admin endpoints disabled")}
 interval,err:=time.ParseDuration(cfg.ScanInterval)
 if err!=nil||interval<time.Minute{return errors.New("SCAN_INTERVAL must be at least 1m")}
 a:=&App{cfg:cfg,bin:Binance{Base:cfg.BinanceBase,Client:&http.Client{Timeout:12*time.Second}}}
 a.load()
 go func(){
  a.scan(context.Background())
  t:=time.NewTicker(interval);defer t.Stop()
  for range t.C{a.scan(context.Background())}
 }()
 mux:=http.NewServeMux()
 mux.HandleFunc("GET /api/health",func(w http.ResponseWriter,r *http.Request){
  a.mu.RLock();defer a.mu.RUnlock()
  jsonReply(w,200,map[string]any{"ok":true,"count":len(a.markets),"scanning":a.scanning,"updatedAt":a.scanned})
 })
 mux.HandleFunc("GET /api/markets",func(w http.ResponseWriter,r *http.Request){
  a.mu.RLock();data:=append([]Market(nil),a.markets...);updated:=a.scanned;a.mu.RUnlock()
  q:=r.URL.Query()
  maxRSI:=number(q.Get("maxRsi"),100)
  maxDist:=number(q.Get("maxDistance"),1e9)
  minVolume:=number(q.Get("minVolume"),500000)
  search:=strings.ToUpper(strings.TrimSpace(q.Get("q")))
  out:=[]Market{}
  for _,m:=range data{
   if m.Volume24h>=minVolume&&m.RSI4h<=maxRSI&&m.Distance90d<=maxDist&&strings.Contains(m.Symbol,search){out=append(out,m)}
  }
  limit:=int(number(q.Get("limit"),100));if limit<1{limit=1};if limit>500{limit=500}
  if len(out)>limit{out=out[:limit]}
  jsonReply(w,200,map[string]any{"data":out,"count":len(out),"updatedAt":updated,"note":"90-day low based on Binance futures candles, not all-time low"})
 })
 mux.HandleFunc("POST /api/admin/scan",func(w http.ResponseWriter,r *http.Request){
  if !a.authorized(r){jsonReply(w,401,map[string]string{"error":"unauthorized"});return}
  go a.scan(context.Background())
  jsonReply(w,202,map[string]string{"message":"scan requested"})
 })
 mux.HandleFunc("GET /api/admin/subscriptions",func(w http.ResponseWriter,r *http.Request){
  if !a.authorized(r){http.Error(w,"unauthorized",401);return}
  a.mu.RLock();defer a.mu.RUnlock()
  jsonReply(w,200,a.subscribers)
 })
 mux.HandleFunc("POST /api/admin/subscriptions",func(w http.ResponseWriter,r *http.Request){
  if !a.authorized(r){http.Error(w,"unauthorized",401);return}
  var s Subscription
  r.Body=http.MaxBytesReader(w,r.Body,4096)
  if json.NewDecoder(r.Body).Decode(&s)!=nil||!strings.Contains(s.Email,"@")||len(s.Email)>254||s.MaxRSI<0||s.MaxRSI>100||s.MaxDistance<0||s.MinVolume<0{
   http.Error(w,"invalid subscription",400);return
  }
  s.ID=time.Now().UnixNano();s.Active=true
  a.mu.Lock();a.subscribers=append(a.subscribers,s);a.mu.Unlock();a.save()
  jsonReply(w,201,s)
 })
 mux.HandleFunc("DELETE /api/admin/subscriptions/{id}",func(w http.ResponseWriter,r *http.Request){
  if !a.authorized(r){http.Error(w,"unauthorized",401);return}
  id,e:=strconv.ParseInt(r.PathValue("id"),10,64)
  if e!=nil{http.Error(w,"invalid id",400);return}
  a.mu.Lock();out:=[]Subscription{}
  for _,s:=range a.subscribers{if s.ID!=id{out=append(out,s)}}
  a.subscribers=out;a.mu.Unlock();a.save()
  w.WriteHeader(204)
 })
 mux.HandleFunc("GET /",func(w http.ResponseWriter,r *http.Request){
  if r.URL.Path!="/"{http.NotFound(w,r);return}
  b,e:=assets.ReadFile("web/index.html");if e!=nil{http.Error(w,"not found",404);return}
  w.Header().Set("Content-Type","text/html; charset=utf-8");_,_ = w.Write(b)
 })
 mux.HandleFunc("GET /app.js",func(w http.ResponseWriter,r *http.Request){
  b,_:=assets.ReadFile("web/app.js");w.Header().Set("Content-Type","text/javascript; charset=utf-8");_,_ = w.Write(b)
 })
 mux.HandleFunc("GET /style.css",func(w http.ResponseWriter,r *http.Request){
  b,_:=assets.ReadFile("web/style.css");w.Header().Set("Content-Type","text/css; charset=utf-8");_,_ = w.Write(b)
 })
 log.Printf("listening %s",cfg.Addr)
 return http.ListenAndServe(cfg.Addr,http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  w.Header().Set("X-Content-Type-Options","nosniff")
  mux.ServeHTTP(w,r)
 }))
}
func number(v string,d float64)float64{
 if v==""{return d}
 f,e:=strconv.ParseFloat(v,64)
 if e!=nil||math.IsNaN(f)||math.IsInf(f,0){return d}
 return f
}
func(a *App) scan(ctx context.Context){
 a.mu.Lock()
 if a.scanning{a.mu.Unlock();return}
 a.scanning=true;a.mu.Unlock()
 defer func(){a.mu.Lock();a.scanning=false;a.mu.Unlock()}()
 ctx,cancel:=context.WithTimeout(ctx,15*time.Minute);defer cancel()
 syms,err:=a.bin.Symbols(ctx)
 if err!=nil{log.Printf("symbols: %v",err);return}
 ticks,err:=a.bin.Tickers(ctx)
 if err!=nil{log.Printf("tickers: %v",err);return}
 funding,err:=a.bin.Funding(ctx)
 if err!=nil{log.Printf("funding: %v",err);return}
 jobs:=make(chan string)
 results:=make(chan Market,len(syms))
 var wg sync.WaitGroup
 for i:=0;i<4;i++{
  wg.Add(1)
  go func(){
   defer wg.Done()
   for s:=range jobs{
    t,ok:=ticks[s]
    if !ok||t.Price<=0||t.Volume<500000{continue}
    c4,_,e:=a.bin.Candles(ctx,s,"4h",120);if e!=nil{continue}
    cd,low,e:=a.bin.Candles(ctx,s,"1d",91);if e!=nil{continue}
    if len(low)>1{low=low[:len(low)-1]}
    r4,r1:=RSI(c4,14),RSI(cd,14)
    d:=Distance(t.Price,Min(low))
    if math.IsNaN(r4)||math.IsNaN(r1)||math.IsNaN(d){continue}
    m:=Market{Symbol:s,Price:t.Price,Volume24h:t.Volume,RSI4h:r4,RSI1d:r1,Low90d:Min(low),Distance90d:d,FundingRate:funding[s],Score:Score(r4,r1,d,t.Volume),UpdatedAt:time.Now().UTC()}
    select{case results<-m:case <-ctx.Done():return}
   }
  }()
 }
 go func(){defer close(jobs);for _,s:=range syms{select{case jobs<-s:case <-ctx.Done():return}}}()
 wg.Wait();close(results)
 arr:=[]Market{}
 for m:=range results{arr=append(arr,m)}
 if len(arr)==0{log.Print("scan returned zero rows; keeping previous data");return}
 sort.Slice(arr,func(i,j int)bool{if arr[i].Score==arr[j].Score{return arr[i].Symbol<arr[j].Symbol};return arr[i].Score>arr[j].Score})
 a.mu.Lock();a.markets=arr;a.scanned=time.Now().UTC();subs:=append([]Subscription(nil),a.subscribers...);a.mu.Unlock()
 log.Printf("scan complete: %d markets",len(arr))
 a.save();a.notify(subs,arr)
}
func(a *App) notify(subs []Subscription,markets []Market){
 if a.cfg.SMTPHost==""||a.cfg.SMTPFrom==""{return}
 for _,s:=range subs{
  if !s.Active{continue}
  matches:=[]Market{}
  for _,m:=range markets{
   if m.RSI4h<=s.MaxRSI&&m.Distance90d<=s.MaxDistance&&m.Volume24h>=s.MinVolume{
    matches=append(matches,m);if len(matches)==15{break}
   }
  }
  if len(matches)==0{continue}
  var body strings.Builder
  body.WriteString("Crypto Screener - matching markets\n\n")
  for _,m:=range matches{
   fmt.Fprintf(&body,"%s price=%.8g RSI4h=%.1f RSI1d=%.1f dist90d=%.2f%% volume=$%.0f funding=%.4f%%\n",m.Symbol,m.Price,m.RSI4h,m.RSI1d,m.Distance90d,m.Volume24h,m.FundingRate*100)
  }
  body.WriteString("\nInformational data only; not investment advice.\n")
  msg:="From: "+a.cfg.SMTPFrom+"\r\nTo: "+s.Email+"\r\nSubject: Crypto Screener report\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n"+body.String()
  var auth smtp.Auth
  if a.cfg.SMTPUser!=""{auth=smtp.PlainAuth("",a.cfg.SMTPUser,a.cfg.SMTPPass,a.cfg.SMTPHost)}
  if err:=smtp.SendMail(a.cfg.SMTPHost+":"+a.cfg.SMTPPort,auth,a.cfg.SMTPFrom,[]string{s.Email},[]byte(msg));err!=nil{
   log.Printf("email to %s: %v",s.Email,err)
  }
 }
}
