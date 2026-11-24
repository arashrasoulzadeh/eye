package service

import (
 "context"
 "encoding/json"
 "fmt"
 "sort"
 "time"
 "github.com/arashrasoulzadeh/eye/internal/model"
 "gorm.io/gorm"
)

type DimensionInput struct {
 Tenant string `json:"tenant_id"`
 From int64 `json:"from_ts"`
 To int64 `json:"to_ts"`
 Types []string `json:"types"`
 Group string `json:"group"`
 Split string `json:"split"`
 Metric string `json:"metric"`
 Operation string `json:"operation"`
 Explode bool `json:"explode"`
}
type DimensionRow struct {
 Group string `json:"group"`
 Split string `json:"split"`
 Value float64 `json:"value"`
 Count int `json:"count"`
}
type DimensionResult struct {
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
 if in.Tenant==""||in.From<0||in.To<in.From||len(in.Types)>100{return out,fmt.Errorf("invalid dimension scope")}
 switch in.Operation {case "count","sum","avg","min","max","distinct":default:return out,fmt.Errorf("invalid operation")}
 ctx,cancel:=context.WithTimeout(ctx,8*time.Second);defer cancel()
 type bucket struct{row DimensionRow; n int; distinct map[string]bool}
 buckets:=map[[2]string]*bucket{}
 keys,types:=map[string]bool{},map[string]bool{}
 err:=q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB)error{
 if q.SetTenantContext!=nil{if err:=q.SetTenantContext(tx,in.Tenant);err!=nil{return err}}
 base:=tx.Model(&model.Event{}).Where("tenant_id = ? AND client_ts >= ? AND client_ts <= ?",in.Tenant,in.From,in.To)
 if err:=base.Session(&gorm.Session{}).Distinct("event_name").Limit(1000).Pluck("event_name",&out.Types).Error;err!=nil{return err}
 if len(in.Types)>0{base=base.Where("event_name IN ?",in.Types)}
 rows,err:=base.Select("event_name,dims").Order("client_ts DESC").Limit(20001).Rows();if err!=nil{return err};defer rows.Close()
 for rows.Next(){
 if out.Scanned==20000{out.Truncated=true;break};out.Scanned++
 var typ,raw string;if err:=rows.Scan(&typ,&raw);err!=nil{return err};types[typ]=true
 var dims map[string]any;if json.Unmarshal([]byte(raw),&dims)!=nil{continue}
 for k:=range dims{if len(keys)<2000{keys[k]=true}}
 if in.Group==""{continue}
 groups:=dimensionValues(dims[in.Group],in.Explode);splits:=[]string{""};if in.Split!=""{if in.Split=="@event"{splits=[]string{typ}}else{splits=dimensionValues(dims[in.Split],in.Explode)}}
 if len(groups)*len(splits)>1000{out.Truncated=true;continue}
 for _,g:=range groups{for _,s:=range splits{
 k:=[2]string{g,s};b:=buckets[k];if b==nil{if len(buckets)>=5000{out.Truncated=true;continue};b=&bucket{row:DimensionRow{Group:g,Split:s},distinct:map[string]bool{}};buckets[k]=b};b.row.Count++
 if in.Operation=="count"{b.row.Value++;continue}
 if in.Operation=="distinct"{for _,v:=range dimensionValues(dims[in.Metric],in.Explode){if dims[in.Metric]!=nil{b.distinct[v]=true}};continue}
 v,ok:=dims[in.Metric].(float64);if !ok{continue};b.n++
 switch in.Operation{case "sum","avg":b.row.Value+=v;case "min":if b.n==1||v<b.row.Value{b.row.Value=v};case "max":if b.n==1||v>b.row.Value{b.row.Value=v}}
 }}
 };return rows.Err()
 })
 for k:=range keys{out.Keys=append(out.Keys,k)};sort.Strings(out.Keys);sort.Strings(out.Types)
 for _,b:=range buckets{if in.Operation=="avg"&&b.n>0{b.row.Value/=float64(b.n)};if in.Operation=="distinct"{b.row.Value=float64(len(b.distinct))};out.Rows=append(out.Rows,b.row)}
 sort.Slice(out.Rows,func(i,j int)bool{a,b:=out.Rows[i],out.Rows[j];if a.Value!=b.Value{return a.Value>b.Value};if a.Group!=b.Group{return a.Group<b.Group};return a.Split<b.Split})
 if len(out.Rows)>1000{out.Rows=out.Rows[:1000];out.Truncated=true};return out,err
}
