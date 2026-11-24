package service
import ("testing";"reflect")
func TestDimensionValues(t *testing.T){
 if got:=dimensionValues([]any{"a","b","a"},true);!reflect.DeepEqual(got,[]string{"a","b"}){t.Fatal(got)}
 if got:=dimensionValues(nil,true);!reflect.DeepEqual(got,[]string{"(missing)"}){t.Fatal(got)}
}

func TestNestedDimensionPaths(t *testing.T){
 got:=flattenDimensions(map[string]any{"device":map[string]any{"os":"Android"},"a/b":42.0,"a~b":true},"",0)
 if got["/device/os"]!="Android"||got["/a~1b"]!=42.0||got["/a~0b"]!=true{t.Fatal(got)}
}
func TestDimensionFilters(t *testing.T){
 dims:=map[string]any{"tags":[]any{"Math","Science"},"score":0.0}
 for _,c:=range []struct{f DimensionFilter;want bool}{
 {DimensionFilter{Key:"tags",Mode:"contains",Value:"math"},true},
 {DimensionFilter{Key:"tags",Mode:"equal",Value:"Science"},true},
 {DimensionFilter{Key:"tags",Mode:"not_equal",Value:"Science"},false},
 {DimensionFilter{Key:"score",Mode:"exists"},true},
 {DimensionFilter{Key:"unknown",Mode:"missing"},true},
 }{if got:=matchDimensionFilter(dims,c.f);got!=c.want{t.Errorf("%+v: %v",c.f,got)}}
}
