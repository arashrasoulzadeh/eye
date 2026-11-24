package handler

import (
 "encoding/json"
 "net/http"
 "github.com/arashrasoulzadeh/eye/internal/service"
)

type QueryExplorerHandler struct { Svc *service.QueryService }

func (h *QueryExplorerHandler) ServeHTTP(w http.ResponseWriter,r *http.Request) {
 if r.Method==http.MethodGet {
  properties,err:=h.Svc.ExplorerProperties(r.Context(),r.URL.Query().Get("tenant_id"))
  if err!=nil {http.Error(w,"schema discovery failed",400);return}
  w.Header().Set("Content-Type","application/json");w.Header().Set("Cache-Control","no-store");json.NewEncoder(w).Encode(properties);return
 }
 if r.Method!=http.MethodPost { w.Header().Set("Allow","GET, POST");http.Error(w,"method not allowed",405);return }
 var input struct {
 Tenant string `json:"tenant_id"`
 SQL string `json:"sql"`
 From int64 `json:"from_ts"`
 To int64 `json:"to_ts"`
 }
 decoder:=json.NewDecoder(http.MaxBytesReader(w,r.Body,32768))
 decoder.DisallowUnknownFields()
 if err:=decoder.Decode(&input);err!=nil {http.Error(w,"invalid request",400);return}
 result,err:=h.Svc.Explore(r.Context(),input.Tenant,input.SQL,input.From,input.To)
 if err!=nil {http.Error(w,"Query failed: "+err.Error(),400);return}
 w.Header().Set("Content-Type","application/json")
 w.Header().Set("Cache-Control","no-store")
 json.NewEncoder(w).Encode(result)
}
