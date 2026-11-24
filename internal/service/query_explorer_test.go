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
