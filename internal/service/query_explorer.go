package service

import (
 "context"
 "fmt"
 "regexp"
 "strings"
 "time"
)

type ExplorerResult struct {
 Columns []string `json:"columns"`
 Rows [][]any `json:"rows"`
 Truncated bool `json:"truncated"`
 DurationMS int64 `json:"duration_ms"`
}

var explorerToken = regexp.MustCompile(`(?i)^(?:[a-z_][a-z_0-9]*(?:\.[a-z_][a-z_0-9]*)*|'(?:[^']|'')*'|[0-9]+(?:\.[0-9]+)?|<=|>=|<>|!=|[(),*+/%=<>-])`)
var explorerIdentifier = regexp.MustCompile(`^[a-z_][a-z_0-9]*$`)
var explorerNumber = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)
var explorerFields = map[string]string{"event_id":"event_id","tenant_id":"tenant_id","session_id":"session_id","user_id":"user_id","type":"event_name","event_name":"event_name","timestamp":"client_ts","client_ts":"client_ts","ingest_ts":"ingest_ts","payload":"payload","dims":"dims","source":"source","is_aggregated":"is_aggregated","aggregation_id":"aggregation_id","experiment_id":"experiment_id","variant":"variant"}
var explorerKeywords = map[string]bool{"where":true,"and":true,"or":true,"not":true,"is":true,"null":true,"in":true,"between":true,"like":true,"group":true,"by":true,"having":true,"order":true,"asc":true,"desc":true,"limit":true,"offset":true,"distinct":true,"as":true,"true":true,"false":true}
var explorerFunctions = map[string]bool{"count":true,"sum":true,"avg":true,"min":true,"max":true,"round":true,"lower":true,"upper":true,"coalesce":true,"length":true,"abs":true}

func compileExplorer(input, dialect string) (string, []any, error) {
 bad := func() (string, []any, error) { return "",nil,fmt.Errorf("unsupported query: use one SELECT FROM events with supported expressions") }
 if len(input)>16000 { return bad() }
 input=strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(input),";"))
 var tokens []string
 for input!="" {
  token:=explorerToken.FindString(input)
  if token=="" || len(tokens)>=512 { return bad() }
  tokens=append(tokens,token)
  input=strings.TrimSpace(input[len(token):])
 }
 if len(tokens)<4 || !strings.EqualFold(tokens[0],"select") { return bad() }
 parts:=[]string{"SELECT"}
 args:=[]any{}
 from:=false
 aliases:=map[string]bool{}
 for i:=1;i<len(tokens);i++ {
  raw:=tokens[i]; t:=strings.ToLower(raw)
  if t=="from" {
   if from || i+1>=len(tokens) || !strings.EqualFold(tokens[i+1],"events") { return bad() }
   from=true; i++; parts=append(parts,"FROM scoped_events AS events"); continue
  }
  if t=="select" || t=="union" || t=="join" || t=="intersect" || t=="except" || t=="with" || t=="into" || t=="over" || t=="returning" { return bad() }
  if strings.HasPrefix(raw,"'") {
   args=append(args,strings.ReplaceAll(raw[1:len(raw)-1],"''","'")); parts=append(parts,"?"); continue
  }
  if explorerNumber.MatchString(t) || strings.Contains("(),*+/%=<>-",t) || t=="<=" || t==">=" || t=="!=" || t=="<>" { parts=append(parts,t); continue }
  if i>0 && strings.EqualFold(tokens[i-1],"as") {
   if !explorerIdentifier.MatchString(t) { return bad() }
   aliases[t]=true; parts=append(parts,"\""+t+"\""); continue
  }
  if explorerKeywords[t] { parts=append(parts,strings.ToUpper(t)); continue }
  if i+1<len(tokens) && tokens[i+1]=="(" {
   if !explorerFunctions[t] { return bad() }
   parts=append(parts,strings.ToUpper(t)); continue
  }
  if aliases[t] { parts=append(parts,"\""+t+"\""); continue }
  field:=strings.TrimPrefix(strings.TrimPrefix(t,"events."),"event.")
  if col,ok:=explorerFields[field];ok { parts=append(parts,"events."+col); continue }
  source:=""; key:=field
  if strings.HasPrefix(field,"payload.") { source="payload"; key=strings.TrimPrefix(field,"payload.") }
  if strings.HasPrefix(field,"dims.") { source="dims"; key=strings.TrimPrefix(field,"dims.") }
  if !explorerIdentifier.MatchString(key) { return bad() }
  // All identifiers are validated; only fixed SQL fragments are emitted.
  extract:=func(col string) string {
   if dialect=="postgres" { return "(NULLIF(events."+col+", '')::jsonb -> '"+key+"')" }
   return "json_extract(CASE WHEN json_valid(events."+col+") THEN events."+col+" ELSE '{}' END, '$."+key+"')"
  }
  expr:=extract("payload")
  if source!="" { expr=extract(source) } else { expr="COALESCE("+expr+", "+extract("dims")+")" }
  if dialect=="postgres" {
   numeric:=i+2<len(tokens) && explorerNumber.MatchString(tokens[i+2])
   if i>=2 && tokens[i-1]=="(" { f:=strings.ToLower(tokens[i-2]); numeric=numeric||f=="sum"||f=="avg"||f=="abs"||f=="round" }
   if numeric { expr="(CASE WHEN jsonb_typeof("+expr+") = 'number' THEN ("+expr+" #>> '{}')::numeric END)" } else { expr="("+expr+" #>> '{}')" }
  }
  parts=append(parts,expr)
 }
 if !from { return bad() }
 return strings.Join(parts," "),args,nil
}

func (q *QueryService) Explore(ctx context.Context, tenant, input string, from, to int64) (ExplorerResult,error) {
 result:=ExplorerResult{Rows:[][]any{}}
 if tenant=="" { return result,fmt.Errorf("select a tenant") }
 if from<0 || to<0 || (to>0 && from>to) { return result,fmt.Errorf("invalid time range") }
 sql,args,err:=compileExplorer(input,q.DB.Dialector.Name())
 if err!=nil { return result,err }
 ctx,cancel:=context.WithTimeout(ctx,5*time.Second); defer cancel()
 start:=time.Now()
 err=q.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
  if q.SetTenantContext!=nil { if err:=q.SetTenantContext(tx,tenant);err!=nil{return err} }
  scope:="SELECT event_id, tenant_id, session_id, user_id, event_name, client_ts, ingest_ts, payload, dims, source, is_aggregated, aggregation_id, experiment_id, variant FROM events WHERE tenant_id = ?"
  params:=[]any{tenant}
  if from>0 { scope+=" AND client_ts >= ?";params=append(params,from) }
  if to>0 { scope+=" AND client_ts <= ?";params=append(params,to) }
  params=append(params,args...)
  rows,err:=tx.Raw("WITH scoped_events AS ("+scope+") SELECT * FROM ("+sql+") AS result LIMIT 1001",params...).Rows()
  if err!=nil{return err};defer rows.Close()
  result.Columns,err=rows.Columns();if err!=nil{return err}
  for rows.Next() {
   if len(result.Rows)==1000 {result.Truncated=true;break}
   values:=make([]any,len(result.Columns)); pointers:=make([]any,len(values))
   for i:=range values {pointers[i]=&values[i]}
   if err:=rows.Scan(pointers...);err!=nil{return err}
   for i,v:=range values {if b,ok:=v.([]byte);ok {values[i]=string(b)}}
   result.Rows=append(result.Rows,values)
  }
  return rows.Err()
 })
 result.DurationMS=time.Since(start).Milliseconds()
 return result,err
}
