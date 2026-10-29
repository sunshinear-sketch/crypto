package app

import "math"

func RSI(closes []float64, period int) float64 {
 if len(closes) <= period { return math.NaN() }
 var gain, loss float64
 for i := 1; i <= period; i++ {
  d := closes[i] - closes[i-1]
  if d > 0 { gain += d } else { loss -= d }
 }
 gain /= float64(period)
 loss /= float64(period)
 for i := period + 1; i < len(closes); i++ {
  d := closes[i] - closes[i-1]
  g,l := 0.0,0.0
  if d > 0 { g=d } else { l=-d }
  gain = (gain*float64(period-1)+g)/float64(period)
  loss = (loss*float64(period-1)+l)/float64(period)
 }
 if loss==0 {
  if gain==0 { return 50 }
  return 100
 }
 return 100-100/(1+gain/loss)
}
func Min(v []float64) float64 {
 if len(v)==0 {return math.NaN()}
 m:=v[0]
 for _,x:=range v[1:] { if x<m {m=x} }
 return m
}
func Distance(price,low float64) float64 {
 if low<=0 {return math.NaN()}
 return (price/low-1)*100
}
func Score(rsi4,rsi1,dist,volume float64) float64 {
 score:=0.0
 if rsi4<30 {score+=30} else if rsi4<40 {score+=15}
 if rsi1<35 {score+=25} else if rsi1<45 {score+=10}
 if dist<5 {score+=25} else if dist<15 {score+=15} else if dist<30 {score+=5}
 if volume>=5000000 {score+=20} else if volume>=500000 {score+=10}
 return score
}
