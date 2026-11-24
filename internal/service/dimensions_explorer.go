package service

import (
 "context"
 "encoding/json"
 "fmt"
 "math"
 "strings"
 "sort"
 "time"
 "github.com/arashrasoulzadeh/eye/internal/model"
 "gorm.io/gorm"
)

type DimensionFilter struct { Key string `json:"key"`; Value string `json:"value"`; Mode string `json:"mode"` }
type DimensionInput struct {
 Filters []DimensionFilter `json:"filters"`
 FilterLogic string `json:"filter_logic"`
 DrillGroup *string `json:"drill_group"`
 DrillSplit *string `json:"drill_split"`
 FullRange bool `json:"full_range"`
 Correlate string `json:"correlate"`
 Tenant string `json:"tenant_id"`
 From int64 `json:"from_ts"`
 To int64 `json:"to_ts"`
 Types []string `json:"types"`
 Group string `json:"group"`
 Split string `json:"split"`
 Metric string `json:"metric"`
 Operation string `json:"operation"`
 FilterKey string `json:"filter_key"`
 FilterValue string `json:"filter_value"`
 FilterMode string `json:"filter_mode"`
 TimeBucket string `json:"time_bucket"`
 Explode bool `json:"explode"`
}
type DimensionRow struct {
 Group string `json:"group"`
 Split string `json:"split"`
 Value float64 `json:"value"`
 Count int `json:"count"`
 Valid int `json:"valid"`
}
type DimensionEvent struct { Type string `json:"type"`; Timestamp int64 `json:"timestamp"`; Dims map[string]any `json:"dims"` }
type DimensionResult struct {
 Events []DimensionEvent `json:"events,omitempty"`
 Correlation *float64 `json:"correlation,omitempty"`
 CorrelationN int `json:"correlation_n"`
 Matched int `json:"matched"`
 Types []string `json:"types"`
 Keys []string `json:"keys"`
 Rows []DimensionRow `json:"rows"`
 Scanned int `json:"scanned"`
 Truncated bool `json:"truncated"`
}
func dimensionValues(v any, explode bool) []string {
 if v==nil{return []string{"(missing)"}}
 if a,ok:=v.([]any);ok&&explode {out:=[]string{};seen:=map[string]bool{};for _,x:=range a {for _,s:=range dimensionValues(x,false){if !seen[s]{out=append(out,s);seen[s]=true}}};if len(out)==0{return []string{"(empty)"}};return out}
 if s,ok:=v.(string);ok{return []string{s}}
 b,_:=json.Marshal(v);return []string{string(b)}
}
func (q *QueryService) AnalyzeDimensions(ctx context.Context,in DimensionInput)(DimensionResult,error){
 out:=DimensionResult{Types:[]string{},Keys:[]string{},Rows:[]DimensionRow{}}
 if in.Tenant==""||in.From<0||in.To<in.From||len(in.Types)>100||len(in.Filters)>20{return out,fmt.Errorf("invalid dimension scope")}
 switch in.Operation {case "count","sum","avg","min","max","distinct","missing","median","p90","p95","p99","stddev":default:return out,fmt.Errorf("invalid operation")}
 ctx,cancel:=context.WithTimeout(ctx,8*time.Second);defer cancel()
 type bucket struct{row DimensionRow; n int; distinct map[string]bool; samples []float64; mean,m2 float64}
 sampleCount:=0
 var sx,sy,sxx,syy,sxy float64
 buckets:=map[[2]string]*bucket{}
 keys,types:=map[string]bool{},map[string]bool{}
 err:=q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB)error{
 if q.SetTenantContext!=nil{if err:=q.SetTenantContext(tx,in.Tenant);err!=nil{return err}}
 base:=tx.Model(&model.Event{}).Where("tenant_id = ? AND client_ts >= ? AND client_ts <= ?",in.Tenant,in.From,in.To)
 if err:=base.Session(&gorm.Session{}).Distinct("event_name").Limit(1000).Pluck("event_name",&out.Types).Error;err!=nil{return err}
 if len(in.Types)>0{base=base.Where("event_name IN ?",in.Types)}
 if in.FullRange && in.Group=="@event" && in.Split=="" && in.Operation=="count" && len(in.Filters)==0 && in.FilterKey=="" && in.TimeBucket=="" && in.Correlate=="" && in.DrillGroup==nil {var grouped []struct{EventName string; Count int};if err:=base.Select("event_name, count(*) AS count").Group("event_name").Order("count DESC").Limit(1001).Scan(&grouped).Error;err!=nil{return err};for i,g:=range grouped{if i==1000{out.Truncated=true;break};out.Rows=append(out.Rows,DimensionRow{Group:g.EventName,Value:float64(g.Count),Count:g.Count});out.Matched+=g.Count};out.Scanned=out.Matched;return nil}
 rows,err:=base.Select("event_name,dims,client_ts").Order("client_ts DESC").Limit(20001).Rows();if err!=nil{return err};defer rows.Close()
 for rows.Next(){
 if out.Scanned==20000{out.Truncated=true;break};out.Scanned++
 var typ,raw string;var ts int64;if err:=rows.Scan(&typ,&raw,&ts);err!=nil{return err};types[typ]=true
 var dims map[string]any;if json.Unmarshal([]byte(raw),&dims)!=nil{continue}
 dims=flattenDimensions(dims,"",0)
 for k:=range dims{if len(keys)<2000{keys[k]=true}}
 if in.FilterKey!="" {matched:=false;for _,v:=range dimensionValues(dims[in.FilterKey],true){if v==in.FilterValue{matched=true}};switch in.FilterMode{case "exists":if dims[in.FilterKey]==nil{continue};case "missing":if dims[in.FilterKey]!=nil{continue};case "not_equal":if matched{continue};default:if !matched{continue}}}
 if len(in.Filters)>0{matches:=0;for _,f:=range in.Filters{if matchDimensionFilter(dims,f){matches++}};if (in.FilterLogic=="or"&&matches==0)||(in.FilterLogic!="or"&&matches!=len(in.Filters)){continue}}
 out.Matched++
 if in.Correlate!=""{x,xok:=dims[in.Metric].(float64);y,yok:=dims[in.Correlate].(float64);if xok&&yok{out.CorrelationN++;sx+=x;sy+=y;sxx+=x*x;syy+=y*y;sxy+=x*y}}
 if in.Group==""&&in.TimeBucket==""{continue}
 groups:=dimensionValues(dims[in.Group],in.Explode);if in.Group=="@event"{groups=[]string{typ}};if in.TimeBucket!=""{stamp:=time.UnixMilli(ts).UTC();switch in.TimeBucket{case "hour":groups=[]string{stamp.Format("2006-01-02 15:00")};case "day":groups=[]string{stamp.Format("2006-01-02")};case "month":groups=[]string{stamp.Format("2006-01")};default:return fmt.Errorf("invalid time bucket")}};splits:=[]string{""};if in.Split!=""{if in.Split=="@event"{splits=[]string{typ}}else{splits=dimensionValues(dims[in.Split],in.Explode)}}
 if len(groups)*len(splits)>1000{out.Truncated=true;continue}
 if in.DrillGroup!=nil&&len(out.Events)<100{found:=false;for _,g:=range groups{for _,s:=range splits{if g==*in.DrillGroup&&(in.DrillSplit==nil||s==*in.DrillSplit){found=true}}};if found{out.Events=append(out.Events,DimensionEvent{Type:typ,Timestamp:ts,Dims:dims})}}
 for _,g:=range groups{for _,s:=range splits{
 k:=[2]string{g,s};b:=buckets[k];if b==nil{if len(buckets)>=5000{out.Truncated=true;continue};b=&bucket{row:DimensionRow{Group:g,Split:s},distinct:map[string]bool{}};buckets[k]=b};b.row.Count++
 if in.Operation=="count"{b.row.Value++;continue};if in.Operation=="missing"{if dims[in.Metric]==nil{b.row.Value++};continue}
 if in.Operation=="distinct"{for _,v:=range dimensionValues(dims[in.Metric],in.Explode){if dims[in.Metric]!=nil{b.distinct[v]=true}};continue}
 v,ok:=dims[in.Metric].(float64);if !ok{continue};b.n++;b.row.Valid++
 if in.Operation=="stddev"{delta:=v-b.mean;b.mean+=delta/float64(b.n);b.m2+=delta*(v-b.mean)}
 if in.Operation=="median"||in.Operation=="p90"||in.Operation=="p95"||in.Operation=="p99"{if sampleCount<200000{b.samples=append(b.samples,v);sampleCount++}else{out.Truncated=true}}
 switch in.Operation{case "sum","avg":b.row.Value+=v;case "min":if b.n==1||v<b.row.Value{b.row.Value=v};case "max":if b.n==1||v>b.row.Value{b.row.Value=v}}
 }}
 };return rows.Err()
 })
 n:=float64(out.CorrelationN);den:=math.Sqrt((n*sxx-sx*sx)*(n*syy-sy*sy));if n>1&&den>0{v:=(n*sxy-sx*sy)/den;if !math.IsNaN(v)&&!math.IsInf(v,0){out.Correlation=&v}}
 for k:=range keys{out.Keys=append(out.Keys,k)};sort.Strings(out.Keys);sort.Strings(out.Types)
 for _,b:=range buckets{if in.Operation=="stddev"&&b.n>0{b.row.Value=math.Sqrt(b.m2/float64(b.n))};if len(b.samples)>0{sort.Float64s(b.samples);p:=map[string]float64{"median":.5,"p90":.9,"p95":.95,"p99":.99}[in.Operation];pos:=p*float64(len(b.samples)-1);lo:=int(pos);hi:=int(math.Ceil(pos));b.row.Value=b.samples[lo]+(b.samples[hi]-b.samples[lo])*(pos-float64(lo))};if in.Operation=="avg"&&b.n>0{b.row.Value/=float64(b.n)};if in.Operation=="distinct"{b.row.Value=float64(len(b.distinct))};out.Rows=append(out.Rows,b.row)}
 sort.Slice(out.Rows,func(i,j int)bool{a,b:=out.Rows[i],out.Rows[j];if a.Value!=b.Value{return a.Value>b.Value};if a.Group!=b.Group{return a.Group<b.Group};return a.Split<b.Split})
 if len(out.Rows)>1000{out.Rows=out.Rows[:1000];out.Truncated=true};return out,err
}

 // Nested paths use JSON Pointer escaping to avoid collisions with literal dots.
 func flattenDimensions(dims map[string]any,prefix string,depth int)map[string]any{
 out:=map[string]any{}
 for k,v:=range dims{
  key:=prefix+"/"+strings.ReplaceAll(strings.ReplaceAll(k,"~","~0"),"/","~1")
  out[key]=v
  if depth<5{if child,ok:=v.(map[string]any);ok{for p,x:=range flattenDimensions(child,key,depth+1){out[p]=x}}}
 }
 if depth==0{for k,v:=range dims{if !strings.HasPrefix(k,"/"){out[k]=v}}}
 return out
 }

func matchDimensionFilter(dims map[string]any,f DimensionFilter)bool{
 v:=dims[f.Key];if f.Mode=="exists"{return v!=nil};if f.Mode=="missing"{return v==nil}
 matched:=false;for _,s:=range dimensionValues(v,true){if f.Mode=="contains"{matched=matched||strings.Contains(strings.ToLower(s),strings.ToLower(f.Value))}else{matched=matched||s==f.Value}}
 if f.Mode=="not_equal"{return !matched};return matched
}
