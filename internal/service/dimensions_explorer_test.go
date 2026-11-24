package service
import ("testing";"reflect")
func TestDimensionValues(t *testing.T){
 if got:=dimensionValues([]any{"a","b","a"},true);!reflect.DeepEqual(got,[]string{"a","b"}){t.Fatal(got)}
 if got:=dimensionValues(nil,true);!reflect.DeepEqual(got,[]string{"(missing)"}){t.Fatal(got)}
}
