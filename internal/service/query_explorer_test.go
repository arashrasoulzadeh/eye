package service

import (
 "strings"
 "testing"
)

func TestExplorerScalarCountOrder(t *testing.T) {
 for _, dialect := range []string{"postgres","sqlite"} {
  sql,_,err:=compileExplorer("SELECT count(*) FROM events ORDER BY timestamp DESC LIMIT 100;",dialect)
  if err!=nil {t.Fatal(err)}
  if strings.Contains(sql,"ORDER BY") || !strings.Contains(sql,"LIMIT 100") {t.Fatalf("%s: %s",dialect,sql)}
  grouped,_,err:=compileExplorer("SELECT type, COUNT(*) AS total FROM events GROUP BY type ORDER BY total DESC",dialect)
  if err!=nil || !strings.Contains(grouped,"ORDER BY") {t.Fatalf("grouped: %s %v",grouped,err)}
 }
}

func TestExplorerQuotedStrings(t *testing.T) {
 for _, dialect := range []string{"postgres","sqlite"} {
  for _, query := range []string{`SELECT events.type FROM events where events.type = "click";`, `SELECT events.type FROM events where events.type = 'click';`} {
   sql,args,err:=compileExplorer(query,dialect)
   if err!=nil || len(args)!=1 || args[0]!="click" || !strings.Contains(sql,"= ?") {t.Fatalf("%s: %s %v %v",dialect,sql,args,err)}
  }
  _,args,err:=compileExplorer(`SELECT type FROM events WHERE type = "a""b'; DROP TABLE events; --"`,dialect)
  if err!=nil || len(args)!=1 || args[0]!=`a"b'; DROP TABLE events; --` {t.Fatalf("literal binding: %v %v",args,err)}
  if _,_,err:=compileExplorer(`SELECT type FROM events WHERE type = "click"; DELETE FROM events`,dialect);err==nil {t.Fatal("accepted multiple statements")}
 }
}
