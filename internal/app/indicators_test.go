package app

import (
 "math"
 "testing"
)
func TestRSI(t *testing.T) {
 up:=[]float64{}
 flat:=[]float64{}
 for i:=0;i<30;i++ {up=append(up,float64(i+1));flat=append(flat,2)}
 if RSI(up,14)!=100 {t.Fatal("rising RSI should be 100")}
 if RSI(flat,14)!=50 {t.Fatal("flat RSI should be 50")}
 if !math.IsNaN(RSI([]float64{1,2},14)){t.Fatal("short series should be NaN")}
}
func TestDistance(t *testing.T) {
 if math.Abs(Distance(105,100)-5)>1e-9 {t.Fatal("distance")}
}
